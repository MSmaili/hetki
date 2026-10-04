package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func reportResolverFixture(t *testing.T) reportContextResolver {
	t.Helper()
	return reportContextResolver{
		client: &MockClient{RunLimitedFunc: func(ctx context.Context, limit int, args ...string) (string, error) {
			require.Equal(t, 256, limit)
			require.Equal(t, []string{"-S", "/socket,with,commas", "display-message", "-p", "-t", "%42", "#{pid}|#{start_time}|#{pane_id}|#{pane_pid}|#{pane_dead}"}, args)
			return "10|123456|%42|300|0\n", ctx.Err()
		}},
		process: func(ctx context.Context, pid int) (reportProcess, error) {
			parent := 0
			switch pid {
			case 100:
				parent = 200
			case 200:
				parent = 300
			case 300:
				parent = 10
			case 10:
				parent = 1
			default:
				return reportProcess{}, errors.New("missing process")
			}
			return reportProcess{pid: pid, parent: parent, birth: fmt.Sprint(pid)}, ctx.Err()
		},
	}
}

func TestReportContextResolvesExactPaneAndServerLifetimes(t *testing.T) {
	r := reportResolverFixture(t)
	attachment, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.NoError(t, err)
	require.Equal(t, "%42", attachment.Pane)
	require.Len(t, attachment.Server, 64)
	require.Len(t, attachment.Generation, 64)
	second, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.NoError(t, err)
	require.Equal(t, attachment, second)
	base := r.process
	r.process = func(ctx context.Context, pid int) (reportProcess, error) {
		p, err := base(ctx, pid)
		if pid == 300 {
			p.birth = "respawn"
		}
		return p, err
	}
	respawn, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.NoError(t, err)
	require.NotEqual(t, attachment.Generation, respawn.Generation)
	require.Equal(t, attachment.Server, respawn.Server)
	r.process = func(ctx context.Context, pid int) (reportProcess, error) {
		p, err := base(ctx, pid)
		if pid == 10 {
			p.birth = "server-restart"
		}
		return p, err
	}
	restarted, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.NoError(t, err)
	require.NotEqual(t, attachment.Server, restarted.Server)
}

func TestReportContextRejectsMissingOrUnsafeEnvironment(t *testing.T) {
	for _, value := range []string{"", "plain", "/socket,0,0", "/socket,10,-1", "/socket,10,x", "relative,10,0", "/socket,2147483648,0", "/socket\x00,10,0"} {
		_, _, err := parseReportEnvironment(value, "%42")
		require.Error(t, err)
	}
	for _, pane := range []string{"", "42", "%42;kill-server", "%42\n", "%123456789012345678901"} {
		_, _, err := parseReportEnvironment("/socket,10,0", pane)
		require.Error(t, err)
	}
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	_, err := ResolveReportContext(context.Background())
	require.ErrorContains(t, err, "inside a tmux pane")
}

func TestReportContextRejectsWrongDeadOrChangedPane(t *testing.T) {
	for _, value := range []string{"", "10|123|%42|300|1\n", "11|123|%42|300|0\n", "10|123|%43|300|0\n", "10||%42|300|0\n", "10|123|%42|0|0\n", "10|123|%42|300|0\nextra"} {
		r := reportResolverFixture(t)
		r.client = &MockClient{RunLimitedFunc: func(context.Context, int, ...string) (string, error) { return value, nil }}
		_, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
		require.Error(t, err)
	}
	r := reportResolverFixture(t)
	calls := 0
	r.client = &MockClient{RunLimitedFunc: func(context.Context, int, ...string) (string, error) {
		calls++
		if calls == 1 {
			return "10|123|%42|300|0", nil
		}
		return "10|123|%42|301|0", nil
	}}
	_, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.ErrorContains(t, err, "changed")
}

func TestReportContextRejectsUnrelatedAndChangingProcesses(t *testing.T) {
	r := reportResolverFixture(t)
	_, err := r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 10)
	require.Error(t, err)
	r = reportResolverFixture(t)
	base := r.process
	reads := 0
	r.process = func(ctx context.Context, pid int) (reportProcess, error) {
		p, err := base(ctx, pid)
		if pid == 300 {
			reads++
			if reads > 1 {
				p.birth = "new"
			}
		}
		return p, err
	}
	_, err = r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.ErrorContains(t, err, "lifetime changed")
	r = reportResolverFixture(t)
	r.process = func(ctx context.Context, pid int) (reportProcess, error) {
		return reportProcess{pid: pid, parent: pid, birth: "cycle"}, nil
	}
	_, err = r.resolve(context.Background(), "/socket,with,commas,10,0", "%42", 100)
	require.ErrorContains(t, err, "not inside")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = r.resolve(ctx, "/socket,with,commas,10,0", "%42", 100)
	require.ErrorIs(t, err, context.Canceled)
}

func TestReportProcessReadsCurrentBirthAndParent(t *testing.T) {
	p, err := readReportProcess(context.Background(), os.Getpid())
	require.NoError(t, err)
	require.Equal(t, os.Getpid(), p.pid)
	require.Equal(t, os.Getppid(), p.parent)
	require.NotEmpty(t, p.birth)
	after, err := readReportProcess(context.Background(), os.Getpid())
	require.NoError(t, err)
	require.Equal(t, p, after)
	for _, pid := range []int{0, -1, 1_000_000_000} {
		_, err := readReportProcess(context.Background(), pid)
		require.Error(t, err)
	}
}
