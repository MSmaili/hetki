package tmux

import (
	"testing"

	"github.com/MSmaili/hetki/internal/backend"
	"github.com/stretchr/testify/assert"
)

func TestMapActionsUsesBackendActionTypes(t *testing.T) {
	b := &TmuxBackend{windowBaseIndex: 1, paneBaseIndex: 1}

	actions := []backend.Action{
		backend.CreateSessionAction{Name: "dev", WindowName: "editor", Path: "~/code"},
		backend.RenameSessionAction{Current: "dev", New: "core"},
		backend.SplitPaneAction{Session: "dev", Window: "editor", Path: "~/api"},
		backend.SendKeysAction{Session: "dev", Window: "editor", Pane: 1, Command: "npm test"},
		backend.SelectLayoutAction{Session: "dev", Window: "editor", Layout: "tiled"},
		backend.ZoomPaneAction{Session: "dev", Window: "editor", Pane: 1},
		backend.CreateWindowAction{Session: "dev", Name: "server", Path: "~/srv"},
		backend.RenameWindowAction{Session: "dev", Window: "server", New: "logs"},
		backend.KillSessionAction{Name: "old"},
		backend.KillWindowAction{Session: "dev", Window: "server"},
	}

	assert.Equal(t, []Action{
		CreateSession{Name: "dev", WindowName: "editor", Path: "~/code"},
		RenameSession{Target: "dev", Name: "core"},
		SplitPane{Target: "dev:1", Path: "~/api"},
		SendKeys{Target: "dev:1.2", Keys: "npm test"},
		SelectLayout{Target: "dev:1", Layout: "tiled"},
		ZoomPane{Target: "dev:1.2"},
		CreateWindow{Session: "dev", Name: "server", Path: "~/srv"},
		RenameWindow{Target: "dev:server", Name: "logs"},
		KillSession{Name: "old"},
		KillWindow{Target: "dev:server"},
	}, b.mapActions(actions))
}

func TestResolveWindowIndex(t *testing.T) {
	sessions := []Session{
		{
			Name: "dev",
			Windows: []Window{
				{Name: "editor", Index: 1},
				{Name: "editor", Index: 2},
				{Name: "logs", Index: 3},
			},
		},
	}

	t.Run("resolves numeric index directly", func(t *testing.T) {
		idx, err := resolveWindowIndex(sessions, "dev", "2")
		assert.NoError(t, err)
		assert.Equal(t, 2, idx)
	})

	t.Run("resolves by name for legacy targets", func(t *testing.T) {
		idx, err := resolveWindowIndex(sessions, "dev", "logs")
		assert.NoError(t, err)
		assert.Equal(t, 3, idx)
	})

	t.Run("fails for unknown numeric index", func(t *testing.T) {
		_, err := resolveWindowIndex(sessions, "dev", "99")
		assert.Error(t, err)
	})
}
