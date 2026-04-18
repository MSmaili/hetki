package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunListWorkspaceFormats(t *testing.T) {
	workspaceYAML := `sessions:
  - name: dev
    windows:
      - name: editor
        path: /tmp/editor
        panes:
          - path: /tmp/editor
          - path: /tmp/api
`

	t.Run("flat names are sorted", func(t *testing.T) {
		resetCommandGlobals()
		home := t.TempDir()
		workspacesDir := filepath.Join(home, ".config", "hetki", "workspaces")
		require.NoError(t, os.MkdirAll(workspacesDir, 0755))
		writeWorkspaceFile(t, workspacesDir, "zeta.yaml", workspaceYAML)
		writeWorkspaceFile(t, workspacesDir, "alpha.yaml", workspaceYAML)
		t.Setenv("HOME", home)

		output := captureStdout(t, func() {
			require.NoError(t, runList(nil, []string{"workspaces"}))
		})

		assert.Equal(t, "alpha\nzeta\n", output)
	})

	t.Run("tree format includes stable session and pane output", func(t *testing.T) {
		resetCommandGlobals()
		home := t.TempDir()
		workspacesDir := filepath.Join(home, ".config", "hetki", "workspaces")
		require.NoError(t, os.MkdirAll(workspacesDir, 0755))
		writeWorkspaceFile(t, workspacesDir, "alpha.yaml", workspaceYAML)
		t.Setenv("HOME", home)

		listSessions = true
		listWindows = true
		listPanes = true
		listFormat = "tree"

		output := captureStdout(t, func() {
			require.NoError(t, runList(nil, []string{"workspaces"}))
		})

		assert.Equal(t, "alpha:dev\n└── editor\n    ├── 0\n    └── 1\n", output)
	})

	t.Run("json format matches listed workspaces", func(t *testing.T) {
		resetCommandGlobals()
		home := t.TempDir()
		workspacesDir := filepath.Join(home, ".config", "hetki", "workspaces")
		require.NoError(t, os.MkdirAll(workspacesDir, 0755))
		writeWorkspaceFile(t, workspacesDir, "alpha.yaml", workspaceYAML)
		t.Setenv("HOME", home)

		listSessions = true
		listWindows = true
		listFormat = "json"

		output := captureStdout(t, func() {
			require.NoError(t, runList(nil, []string{"workspaces"}))
		})

		assert.Equal(t, "[\n  {\n    \"name\": \"alpha:dev\",\n    \"windows\": [\n      {\n        \"name\": \"editor\"\n      }\n    ]\n  }\n]\n", output)
	})
}
