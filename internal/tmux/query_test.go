package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListSessionsQuery(t *testing.T) {
	q := ListSessionsQuery{}

	t.Run("args", func(t *testing.T) {
		assert.Equal(t, []string{"list-sessions", "-F", "#{session_name}"}, q.Args())
	})

	tests := []struct {
		name   string
		output string
		want   []Session
	}{
		{"empty", "", []Session{}},
		{"single", "dev", []Session{{Name: "dev"}}},
		{"multiple", "dev\nwork\ntest", []Session{{Name: "dev"}, {Name: "work"}, {Name: "test"}}},
		{"with empty lines", "dev\n\nwork\n", []Session{{Name: "dev"}, {Name: "work"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.Parse(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListWindowsQuery(t *testing.T) {
	q := ListWindowsQuery{Session: "dev"}

	t.Run("args", func(t *testing.T) {
		assert.Equal(t, []string{"list-windows", "-t", "dev", "-F", "#{window_name}|#{pane_current_path}"}, q.Args())
	})

	tests := []struct {
		name   string
		output string
		want   []Window
	}{
		{"empty", "", []Window{}},
		{"single", "editor|~/code", []Window{{Name: "editor", Path: "~/code"}}},
		{"multiple", "editor|~/code\nserver|~/api", []Window{{Name: "editor", Path: "~/code"}, {Name: "server", Path: "~/api"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.Parse(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListPanesQuery(t *testing.T) {
	q := ListPanesQuery{Target: "dev:editor"}

	t.Run("args", func(t *testing.T) {
		assert.Equal(t, []string{"list-panes", "-t", "dev:editor", "-F", "#{pane_current_path}|#{pane_current_command}"}, q.Args())
	})

	tests := []struct {
		name   string
		output string
		want   []Pane
	}{
		{"empty", "", []Pane{}},
		{"single", "~/code|vim", []Pane{{Path: "~/code", Command: "vim"}}},
		{"multiple", "~/code|vim\n~/api|node", []Pane{{Path: "~/code", Command: "vim"}, {Path: "~/api", Command: "node"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.Parse(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
