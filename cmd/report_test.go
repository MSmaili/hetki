package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	appreport "github.com/MSmaili/hetki/internal/app/report"
	"github.com/MSmaili/hetki/internal/report"
	"github.com/stretchr/testify/require"
)

type unreadableReportStdin struct{}

func (unreadableReportStdin) Read([]byte) (int, error) { panic("report must not read stdin") }

func runReportStatus(t *testing.T, run func(context.Context, report.Status) error, ctx context.Context, args ...string) error {
	t.Helper()
	command := newReportCommandWithRunner(run)
	command.SetContext(ctx)
	command.SetIn(unreadableReportStdin{})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	err := command.Execute()
	require.Empty(t, output.String(), "report succeeds quietly and keeps machine internals private")
	return err
}

func stubReportService() appreport.Service {
	return appreport.NewService(func(ctx context.Context) (report.Binding, error) {
		if err := ctx.Err(); err != nil {
			return report.Binding{}, err
		}
		return report.Binding{Server: "test-server", Pane: "%42", Generation: "birth-1"}, nil
	})
}

func TestReportCommandNeedsOnlyStatusAndNeverReadsJSON(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	service := stubReportService()
	for _, status := range report.Statuses() {
		require.NoError(t, runReportStatus(t, service.Run, context.Background(), "--status", status))
	}
	path, err := report.DefaultPath()
	require.NoError(t, err)
	state, err := report.NewStore(path).Load(context.Background())
	require.NoError(t, err)
	require.Len(t, state.Streams, 1)
	require.Equal(t, report.Unknown, state.Streams[0].Activity)
	require.EqualValues(t, 4, state.Streams[0].Sequence)
	require.Empty(t, state.Events, "status is never an implicit completion")
}

func TestReportCommandRejectsInvalidStatusBeforeResolving(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, args := range [][]string{nil, {"--status", ""}, {"--status", "done"}, {"--status", "smthgelse"}, {"--status", "Working"}, {"--status", "working", "unexpected"}, {"--session", "s1", "--status", "working"}} {
		require.Error(t, runReportStatus(t, nil, context.Background(), args...))
	}
	path, err := report.DefaultPath()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Dir(path))
	require.True(t, os.IsNotExist(err))
}

func TestReportCommandPropagatesContextAndAttachmentFailures(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	service := stubReportService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, runReportStatus(t, service.Run, ctx, "--status", "working"), context.Canceled)
	service.ResolveBinding = func(context.Context) (report.Binding, error) {
		return report.Binding{}, errors.New("not inside tmux")
	}
	require.ErrorContains(t, runReportStatus(t, service.Run, context.Background(), "--status", "working"), "not inside tmux")
	path, err := report.DefaultPath()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Dir(path))
	require.True(t, os.IsNotExist(err))
}

func TestReportCommandRegisteredAndCompletesStatuses(t *testing.T) {
	found, _, err := rootCmd.Find([]string{"report"})
	require.NoError(t, err)
	require.Equal(t, "report", found.Name())
	command := newReportCommand()
	completion, ok := command.GetFlagCompletionFunc("status")
	require.True(t, ok)
	values, directive := completion(command, nil, "")
	require.Equal(t, report.Statuses(), values)
	require.NotZero(t, directive)
	for _, internal := range []string{"stream-id", "sequence", "lease", "pane", "session", "registration-id"} {
		require.Nil(t, command.Flags().Lookup(internal))
	}
}
