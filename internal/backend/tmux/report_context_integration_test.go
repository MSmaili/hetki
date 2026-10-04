//go:build integration && (darwin || linux)

package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MSmaili/hetki/internal/report"
	"github.com/stretchr/testify/require"
)

func TestReportStatusCLIInfersPaneAndSeparatesRespawn(t *testing.T) {
	dir := t.TempDir()
	binary := os.Getenv("HETKI_REPORT_CLI_TEST_BINARY")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if binary == "" {
		binary = filepath.Join(dir, "hetki")
		build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
		build.Dir = "../../.."
		out, err := build.CombinedOutput()
		require.NoError(t, err, "%s", out)
	}
	stateHome := filepath.Join(dir, "state")
	marker := filepath.Join(dir, "ready")
	stderr := filepath.Join(dir, "errors")
	b := newIsolatedTmuxBackend(t)
	script := fmt.Sprintf("set -eu\nexport XDG_STATE_HOME=%q\n%q report --status working 2>%q\n%q report --status needs-input 2>>%q\n%q report --status idle 2>>%q\n%q report --status unknown 2>>%q\ntouch %q\nsleep 3600\n", stateHome, binary, stderr, binary, stderr, binary, stderr, binary, stderr, marker)
	_, err := b.client.Run(ctx, "new-session", "-d", "-s", "report", "sh", "-c", script)
	require.NoError(t, err)
	require.Eventually(t, func() bool { _, err := os.Stat(marker); return err == nil }, 5*time.Second, 20*time.Millisecond)
	store := report.NewStore(filepath.Join(stateHome, "hetki", "reports", "state.json"))
	state, err := store.Load(ctx)
	require.NoError(t, err)
	require.Len(t, state.Streams, 1)
	require.Equal(t, report.Unknown, state.Streams[0].Activity)
	require.EqualValues(t, 4, state.Streams[0].Sequence)
	require.Empty(t, state.Events)
	first := state.Streams[0].Binding
	metadata, err := b.client.Run(ctx, "display-message", "-p", "-t", "report:", "#{socket_path},#{pid},0|#{pane_id}")
	require.NoError(t, err)
	fields := strings.Split(strings.TrimSpace(metadata), "|")
	require.Len(t, fields, 2)
	// A host process with copied TMUX variables must not report into this pane.
	outside := exec.CommandContext(ctx, binary, "report", "--status", "working")
	outside.Env = append(os.Environ(), "TMUX="+fields[0], "TMUX_PANE="+fields[1], "XDG_STATE_HOME="+filepath.Join(dir, "outside"))
	out, err := outside.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "not inside the selected tmux pane")
	secondMarker := filepath.Join(dir, "respawned")
	respawn := fmt.Sprintf("set -eu\nexport XDG_STATE_HOME=%q\n%q report --status working 2>>%q\ntouch %q\nsleep 3600\n", stateHome, binary, stderr, secondMarker)
	_, err = b.client.Run(ctx, "respawn-pane", "-k", "-t", fields[1], "sh", "-c", respawn)
	require.NoError(t, err)
	require.Eventually(t, func() bool { _, err := os.Stat(secondMarker); return err == nil }, 5*time.Second, 20*time.Millisecond)
	state, err = store.Load(ctx)
	require.NoError(t, err)
	require.Len(t, state.Streams, 2)
	var found bool
	for _, stream := range state.Streams {
		if stream.Binding.Generation != first.Generation {
			found = true
			require.Equal(t, first.Server, stream.Binding.Server)
			require.Equal(t, first.Pane, stream.Binding.Pane)
			require.Equal(t, report.Working, stream.Activity)
		}
	}
	require.True(t, found, "respawn gets a distinct process-birth generation")
	_, err = b.client.Run(ctx, "kill-server")
	require.NoError(t, err)
	restartMarker := filepath.Join(dir, "server-restarted")
	restart := fmt.Sprintf("set -eu\nexport XDG_STATE_HOME=%q\n%q report --status working 2>>%q\ntouch %q\nsleep 3600\n", stateHome, binary, stderr, restartMarker)
	_, err = b.client.Run(ctx, "new-session", "-d", "-s", "report", "sh", "-c", restart)
	require.NoError(t, err)
	require.Eventually(t, func() bool { _, err := os.Stat(restartMarker); return err == nil }, 5*time.Second, 20*time.Millisecond)
	state, err = store.Load(ctx)
	require.NoError(t, err)
	require.Len(t, state.Streams, 3)
	var restarted bool
	for _, stream := range state.Streams {
		if stream.Binding.Server != first.Server {
			restarted = true
		}
	}
	require.True(t, restarted, "server restart cannot reuse the old server identity")
	errors, err := os.ReadFile(stderr)
	require.NoError(t, err)
	require.Empty(t, errors)
}
