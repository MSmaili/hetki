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
		backend.SplitPaneAction{Session: "dev", Window: "editor", Path: "~/api"},
		backend.SendKeysAction{Session: "dev", Window: "editor", Pane: 1, Command: "npm test"},
		backend.SelectLayoutAction{Session: "dev", Window: "editor", Layout: "tiled"},
		backend.ZoomPaneAction{Session: "dev", Window: "editor", Pane: 1},
		backend.CreateWindowAction{Session: "dev", Name: "server", Path: "~/srv"},
		backend.KillSessionAction{Name: "old"},
		backend.KillWindowAction{Session: "dev", Window: "server"},
	}

	assert.Equal(t, []Action{
		CreateSession{Name: "dev", WindowName: "editor", Path: "~/code"},
		SplitPane{Target: "dev:1", Path: "~/api"},
		SendKeys{Target: "dev:1.2", Keys: "npm test"},
		SelectLayout{Target: "dev:1", Layout: "tiled"},
		ZoomPane{Target: "dev:1.2"},
		CreateWindow{Session: "dev", Name: "server", Path: "~/srv"},
		KillSession{Name: "old"},
		KillWindow{Target: "dev:server"},
	}, b.mapActions(actions))
}
