package report

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MSmaili/hetki/internal/report"
	"github.com/stretchr/testify/require"
)

func TestServiceResolvesThenPersistsWithBoundedContext(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "reports", "state.json")
	binding := report.Binding{Server: "server", Pane: "%1", Generation: "birth"}
	service := NewService(func(ctx context.Context) (report.Binding, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.Positive(t, time.Until(deadline))
		require.LessOrEqual(t, time.Until(deadline), reportingTimeout)
		return binding, nil
	})
	service.StatePath = func() (string, error) { return path, nil }
	require.NoError(t, service.Run(context.Background(), report.StatusNeedsInput))
	state, err := report.NewStore(path).Load(context.Background())
	require.NoError(t, err)
	require.Len(t, state.Streams, 1)
	require.Equal(t, binding, state.Streams[0].Binding)
	require.NotEmpty(t, state.Streams[0].PendingInput)
}

func TestServiceValidatesBeforeResolvingOrWriting(t *testing.T) {
	t.Parallel()
	service := NewService(func(context.Context) (report.Binding, error) {
		t.Fatal("invalid status/canceled context must not resolve attachment")
		return report.Binding{}, nil
	})
	service.StatePath = nil
	require.ErrorContains(t, service.Run(context.Background(), "invalid"), "status must be")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, service.Run(ctx, report.StatusWorking), context.Canceled)
	require.ErrorContains(t, (Service{}).Run(context.Background(), report.StatusWorking), "not configured")
}

func TestServicePreservesErrorsAndDoesNotWriteOnResolutionFailure(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "reports", "state.json")
	want := errors.New("attachment unavailable")
	service := NewService(func(context.Context) (report.Binding, error) { return report.Binding{}, want })
	service.StatePath = func() (string, error) { t.Fatal("attachment must resolve before state path"); return path, nil }
	require.ErrorIs(t, service.Run(context.Background(), report.StatusWorking), want)
	service.ResolveBinding = func(context.Context) (report.Binding, error) {
		return report.Binding{Server: "server", Pane: "%1", Generation: "birth"}, nil
	}
	service.StatePath = func() (string, error) { return "", want }
	require.ErrorIs(t, service.Run(context.Background(), report.StatusWorking), want)
	_, err := os.Stat(filepath.Dir(path))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestServiceHonorsEarlierCallerDeadlineAndCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	callerDeadline, _ := ctx.Deadline()
	path := filepath.Join(t.TempDir(), "reports", "state.json")
	service := NewService(func(ctx context.Context) (report.Binding, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.Equal(t, callerDeadline, deadline)
		cancel()
		return report.Binding{Server: "server", Pane: "%1", Generation: "birth"}, nil
	})
	service.StatePath = func() (string, error) { return path, nil }
	require.ErrorIs(t, service.Run(ctx, report.StatusWorking), context.Canceled)
	_, err := os.Stat(filepath.Dir(path))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestServiceAddsContextToPersistenceFailure(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "public-reports")
	require.NoError(t, os.Mkdir(dir, 0755))
	require.NoError(t, os.Chmod(dir, 0755))
	service := NewService(func(context.Context) (report.Binding, error) {
		return report.Binding{Server: "server", Pane: "%1", Generation: "birth"}, nil
	})
	service.StatePath = func() (string, error) { return filepath.Join(dir, "state.json"), nil }
	require.ErrorContains(t, service.Run(context.Background(), report.StatusWorking), "record report status: report directory must be")
}
