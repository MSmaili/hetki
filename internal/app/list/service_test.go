package list

import (
	"testing"

	"github.com/MSmaili/hetki/internal/backend"
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

func TestServiceRunWorkspacesReturnsSortedNames(t *testing.T) {
	service := Service{
		GetConfigDir: func() (string, error) { return "/config", nil },
		ScanWorkspaces: func(dir string) (map[string]string, error) {
			assert.Equal(t, "/config/workspaces", dir)
			return map[string]string{"zeta": "/zeta.yaml", "alpha": "/alpha.yaml"}, nil
		},
	}

	result, err := service.Run(Options{})
	require.NoError(t, err)
	assert.True(t, result.NamesOnly)
	assert.Equal(t, []string{"alpha", "zeta"}, result.Names)
	assert.Empty(t, result.Items)
}

func TestServiceRunSessionsReturnsWindowAndPaneState(t *testing.T) {
	stub := &stubBackend{queryResult: backend.StateResult{
		Sessions: []backend.Session{{
			Name: "dev",
			Windows: []backend.Window{{
				Name:  "editor",
				Panes: []backend.Pane{{Index: 0}, {Index: 1}},
			}},
		}},
		Active: backend.ActiveContext{Session: "dev", Window: "editor", Pane: 1},
	}}

	service := NewService(func(...string) (backend.Backend, error) { return stub, nil })
	result, err := service.Run(Options{Mode: ModeSessions, IncludeWindows: true, IncludePanes: true})
	require.NoError(t, err)
	assert.False(t, result.NamesOnly)
	assert.Equal(t, []Item{{
		Name: "dev",
		Windows: []Window{{
			Name:       "editor",
			Panes:      []int{0, 1},
			ActivePane: 1,
		}},
	}}, result.Items)
}
