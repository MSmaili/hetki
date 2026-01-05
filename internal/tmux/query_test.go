package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadStateQuery(t *testing.T) {
	q := LoadStateQuery{}

	t.Run("args", func(t *testing.T) {
		expected := []string{
			"list-panes", "-a", "-F",
			"#{session_id}|#{session_name}|#{window_name}|#{pane_current_path}|#{pane_current_command}|#{TMS_WORKSPACE_PATH}",
			";", "show-options", "-gv", "pane-base-index",
		}
		assert.Equal(t, expected, q.Args())
	})

	tests := []struct {
		name   string
		output string
		want   LoadStateResult
	}{
		{"empty", "", LoadStateResult{}},
		{
			name:   "single session single window single pane",
			output: "$1|dev|editor|~/code|vim|/path/to/workspace.yaml\n0",
			want: LoadStateResult{
				Sessions: []Session{{
					Name:          "dev",
					WorkspacePath: "/path/to/workspace.yaml",
					Windows: []Window{{
						Name:  "editor",
						Path:  "~/code",
						Panes: []Pane{{Path: "~/code", Command: "vim"}},
					}},
				}},
				PaneBaseIndex: 0,
			},
		},
		{
			name:   "multiple panes same window",
			output: "$1|dev|editor|~/code|vim|\n$1|dev|editor|~/api|node|\n1",
			want: LoadStateResult{
				Sessions: []Session{{
					Name: "dev",
					Windows: []Window{{
						Name:  "editor",
						Path:  "~/code",
						Panes: []Pane{{Path: "~/code", Command: "vim"}, {Path: "~/api", Command: "node"}},
					}},
				}},
				PaneBaseIndex: 1,
			},
		},
		{
			name:   "multiple windows",
			output: "$1|dev|editor|~/code|vim|/ws.yaml\n$1|dev|server|~/api|node|/ws.yaml\n1",
			want: LoadStateResult{
				Sessions: []Session{{
					Name:          "dev",
					WorkspacePath: "/ws.yaml",
					Windows: []Window{
						{Name: "editor", Path: "~/code", Panes: []Pane{{Path: "~/code", Command: "vim"}}},
						{Name: "server", Path: "~/api", Panes: []Pane{{Path: "~/api", Command: "node"}}},
					},
				}},
				PaneBaseIndex: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.Parse(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPaneBaseIndexQuery(t *testing.T) {
	q := PaneBaseIndexQuery{}

	t.Run("args", func(t *testing.T) {
		assert.Equal(t, []string{"show-options", "-gv", "pane-base-index"}, q.Args())
	})

	tests := []struct {
		name   string
		output string
		want   int
	}{
		{"empty", "", 0},
		{"zero", "0", 0},
		{"one", "1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.Parse(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
