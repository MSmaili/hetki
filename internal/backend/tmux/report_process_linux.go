package tmux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func readReportProcess(ctx context.Context, pid int) (reportProcess, error) {
	if err := ctx.Err(); err != nil {
		return reportProcess{}, err
	}
	if pid <= 0 || pid > math.MaxInt32 {
		return reportProcess{}, errors.New("invalid reporting process PID")
	}
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/stat", pid), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return reportProcess{}, fmt.Errorf("open reporting process metadata: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return reportProcess{}, fmt.Errorf("read reporting process metadata: %w", err)
	}
	if len(data) > 4096 {
		return reportProcess{}, errors.New("cannot read bounded reporting process metadata")
	}
	if err := ctx.Err(); err != nil {
		return reportProcess{}, err
	}
	return parseReportProcessStat(string(data), pid)
}

func parseReportProcessStat(value string, pid int) (reportProcess, error) {
	open, close := strings.IndexByte(value, '('), strings.LastIndexByte(value, ')')
	if open < 1 || close <= open {
		return reportProcess{}, errors.New("invalid reporting process metadata")
	}
	observed, err := strconv.Atoi(strings.TrimSpace(value[:open]))
	fields := strings.Fields(value[close+1:])
	if err != nil || observed != pid || len(fields) < 20 {
		return reportProcess{}, errors.New("invalid reporting process metadata")
	}
	parent, parentErr := strconv.Atoi(fields[1])
	birth, birthErr := strconv.ParseUint(fields[19], 10, 64)
	if parentErr != nil || birthErr != nil || parent < 0 || parent > math.MaxInt32 || birth == 0 || len(fields[0]) != 1 || fields[0] == "Z" || fields[0] == "X" || fields[0] == "x" {
		return reportProcess{}, errors.New("reporting process is invalid or exiting")
	}
	return reportProcess{pid: pid, parent: parent, birth: fields[19]}, nil
}
