package tmux

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ReportContext identifies the invoking terminal's current pane lifetime. It
// is a best-effort local observation, not agent-session association or authority.
type ReportContext struct{ Server, Pane, Generation string }

type reportProcess struct {
	pid, parent int
	birth       string
}

type reportContextResolver struct {
	client  Client
	process func(context.Context, int) (reportProcess, error)
}

var reportPaneID = regexp.MustCompile(`^%[0-9]{1,20}$`)

// ResolveReportContext never picks an active/default pane when context is absent.
// The invoking process must belong to the explicitly selected pane's ancestry.
func ResolveReportContext(ctx context.Context) (ReportContext, error) {
	if err := ctx.Err(); err != nil {
		return ReportContext{}, err
	}
	value, pane := os.Getenv("TMUX"), os.Getenv("TMUX_PANE")
	if value == "" || pane == "" {
		return ReportContext{}, errors.New("report must run inside a tmux pane (TMUX and TMUX_PANE are required)")
	}
	c, err := New()
	if err != nil {
		return ReportContext{}, err
	}
	return (reportContextResolver{client: c, process: readReportProcess}).resolve(ctx, value, pane, os.Getpid())
}

func parseReportEnvironment(value, pane string) (string, int, error) {
	// TMUX is socket,pid,client-index. Split from the right: socket filenames
	// can contain commas. Never pass these values through a shell.
	last := strings.LastIndexByte(value, ',')
	if last < 0 {
		return "", 0, errors.New("invalid tmux reporting context")
	}
	previous := strings.LastIndexByte(value[:last], ',')
	if previous < 0 {
		return "", 0, errors.New("invalid tmux reporting context")
	}
	socket := value[:previous]
	pid, err := strconv.Atoi(value[previous+1 : last])
	index, indexErr := strconv.Atoi(value[last+1:])
	if err != nil || indexErr != nil || index < 0 || pid <= 0 || pid > math.MaxInt32 ||
		!filepath.IsAbs(socket) || len(socket) > 4096 || strings.ContainsRune(socket, 0) || !reportPaneID.MatchString(pane) {
		return "", 0, errors.New("invalid tmux reporting context")
	}
	return socket, pid, nil
}

type reportPane struct {
	serverPID   int
	serverStart string
	pane        string
	panePID     int
}

func parseReportPane(value, wanted string, serverPID int) (reportPane, error) {
	fields := strings.Split(strings.TrimSuffix(value, "\n"), "|")
	if len(fields) != 5 {
		return reportPane{}, errors.New("invalid tmux report attachment response")
	}
	pid, err := strconv.Atoi(fields[0])
	start, startErr := strconv.ParseUint(fields[1], 10, 64)
	panePID, paneErr := strconv.Atoi(fields[3])
	if err != nil || startErr != nil || start == 0 || paneErr != nil || panePID <= 0 || panePID > math.MaxInt32 ||
		pid != serverPID || fields[2] != wanted || fields[4] != "0" {
		return reportPane{}, errors.New("tmux pane is missing, dead or belongs to a different server")
	}
	return reportPane{serverPID: pid, serverStart: fields[1], pane: fields[2], panePID: panePID}, nil
}

func (r reportContextResolver) resolve(ctx context.Context, environment, pane string, callerPID int) (ReportContext, error) {
	if err := ctx.Err(); err != nil {
		return ReportContext{}, err
	}
	socket, pid, err := parseReportEnvironment(environment, pane)
	if err != nil {
		return ReportContext{}, err
	}
	query := func() (reportPane, error) {
		out, err := r.client.RunLimited(ctx, 256, "-S", socket, "display-message", "-p", "-t", pane,
			"#{pid}|#{start_time}|#{pane_id}|#{pane_pid}|#{pane_dead}")
		if err != nil {
			return reportPane{}, fmt.Errorf("resolve tmux report attachment: %w", err)
		}
		return parseReportPane(out, pane, pid)
	}
	before, err := query()
	if err != nil {
		return ReportContext{}, err
	}
	server, err := r.process(ctx, pid)
	if err != nil {
		return ReportContext{}, err
	}
	root, err := r.process(ctx, before.panePID)
	if err != nil {
		return ReportContext{}, err
	}
	current := callerPID
	seen := make(map[int]bool)
	belongs := false
	for depth := 0; depth < 128 && current > 0 && !seen[current]; depth++ {
		if current == root.pid {
			belongs = true
			break
		}
		seen[current] = true
		proc, err := r.process(ctx, current)
		if err != nil {
			return ReportContext{}, err
		}
		current = proc.parent
	}
	if !belongs {
		return ReportContext{}, errors.New("reporting process is not inside the selected tmux pane; run the hook in the local terminal client")
	}
	after, err := query()
	if err != nil {
		return ReportContext{}, err
	}
	if before != after {
		return ReportContext{}, errors.New("tmux attachment changed during reporting")
	}
	serverAfter, err := r.process(ctx, pid)
	if err != nil {
		return ReportContext{}, err
	}
	rootAfter, err := r.process(ctx, root.pid)
	if err != nil {
		return ReportContext{}, err
	}
	if server.birth != serverAfter.birth || root.birth != rootAfter.birth {
		return ReportContext{}, errors.New("tmux process lifetime changed during reporting")
	}
	if err := ctx.Err(); err != nil {
		return ReportContext{}, err
	}
	// Process lifetime identifies the server independently of socket spelling
	// (for example /tmp versus /private/tmp aliases).
	return ReportContext{Server: reportContextHash(strconv.Itoa(pid), before.serverStart, server.birth), Pane: pane,
		Generation: reportContextHash(strconv.Itoa(root.pid), root.birth)}, nil
}

func reportContextHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
