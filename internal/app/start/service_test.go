package start

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/MSmaili/hetki/internal/logger"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBackend struct {
	queryResult backend.StateResult
	queryErr    error
	dryRunLines []string
	applyErr    error

	applyCalls  int
	attachCalls int
	dryRunCalls int
}

func (s *stubBackend) Name() string { return "stub" }

func (s *stubBackend) QueryState() (backend.StateResult, error) {
	if s.queryErr != nil {
		return backend.StateResult{}, s.queryErr
	}
	return s.queryResult, nil
}

func (s *stubBackend) Apply(actions []backend.Action) error {
	s.applyCalls++
	return s.applyErr
}

func (s *stubBackend) DryRun(actions []backend.Action) []string {
	s.dryRunCalls++
	return append([]string(nil), s.dryRunLines...)
}

func (s *stubBackend) Attach(session string) error {
	s.attachCalls++
	return nil
}

func (s *stubBackend) Switch(target string) error { return nil }

func TestServiceRunDryRunOutputsPlan(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := "sessions:\n  - name: dev\n    windows:\n      - name: editor\n        path: " + filepath.ToSlash(tmpDir) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hetki.yaml"), []byte(workspace), 0644))

	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previousWD))
	})

	stub := &stubBackend{dryRunLines: []string{"tmux new-session -d -s dev -n editor -c " + filepath.ToSlash(tmpDir)}}
	service := NewService(func(...string) (backend.Backend, error) { return stub, nil })

	output := captureLoggerOutput(t, func() {
		require.NoError(t, service.Run(Options{DryRun: true}))
	})

	assert.Contains(t, output, "Dry run - actions to execute:")
	assert.Contains(t, output, stub.dryRunLines[0])
	assert.Equal(t, 1, stub.dryRunCalls)
	assert.Zero(t, stub.applyCalls)
	assert.Zero(t, stub.attachCalls)
}

func TestServiceRunFailsWhenBackendStateQueryFails(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := "sessions:\n  - name: dev\n    windows:\n      - name: editor\n        path: " + filepath.ToSlash(tmpDir) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hetki.yaml"), []byte(workspace), 0644))

	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previousWD))
	})

	stub := &stubBackend{queryErr: errors.New("query failed")}
	service := NewService(func(...string) (backend.Backend, error) { return stub, nil })

	err = service.Run(Options{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to query backend state: query failed")
	assert.ErrorContains(t, err, "muxie list sessions")
	assert.Zero(t, stub.dryRunCalls)
	assert.Zero(t, stub.applyCalls)
}

func captureLoggerOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	previousOutput := logger.SetOutput(&buf)
	previousNoColor := color.NoColor
	color.NoColor = true
	defer func() {
		logger.SetOutput(previousOutput)
		color.NoColor = previousNoColor
	}()

	fn()
	return buf.String()
}
