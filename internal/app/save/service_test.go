package save

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/MSmaili/hetki/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBackend struct {
	queryResult backend.StateResult
	queryErr    error
}

func (s *stubBackend) Name() string { return "stub" }

func (s *stubBackend) QueryState() (backend.StateResult, error) {
	if s.queryErr != nil {
		return backend.StateResult{}, s.queryErr
	}
	return s.queryResult, nil
}

func (s *stubBackend) Apply(actions []backend.Action) error     { return nil }
func (s *stubBackend) DryRun(actions []backend.Action) []string { return nil }
func (s *stubBackend) Attach(session string) error              { return nil }
func (s *stubBackend) Switch(target string) error               { return nil }

func TestServiceRunWritesCurrentSessionWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	outputPath := filepath.Join(t.TempDir(), "workspace.yaml")
	stub := &stubBackend{queryResult: backend.StateResult{
		Sessions: []backend.Session{{
			Name: "dev",
			Windows: []backend.Window{{
				Name: "editor",
				Path: filepath.Join(home, "code", "hetki"),
			}},
		}},
		Active: backend.ActiveContext{Session: "dev"},
	}}

	service := NewService(func(...string) (backend.Backend, error) { return stub, nil })
	path, err := service.Run(Options{Path: outputPath})
	require.NoError(t, err)
	assert.Equal(t, outputPath, path)

	loader := manifest.NewFileLoader(outputPath)
	workspace, err := loader.Load()
	require.NoError(t, err)
	require.Len(t, workspace.Sessions, 1)
	assert.Equal(t, "dev", workspace.Sessions[0].Name)
	require.Len(t, workspace.Sessions[0].Windows, 1)
	assert.Equal(t, filepath.Join(home, "code", "hetki"), workspace.Sessions[0].Windows[0].Path)

	contents, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "path: ~/code/hetki")
}
