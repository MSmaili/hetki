package core

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Quit        []string
	Up          []string
	Down        []string
	PageUp      []string
	PageDown    []string
	Search      []string
	ClearFilter []string
	Backspace   []string
	Cancel      []string
	Confirm     []string
	Refresh     []string
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:        []string{"q", "ctrl+c"},
		Up:          []string{"up", "k"},
		Down:        []string{"down", "j"},
		PageUp:      []string{"pgup", "u"},
		PageDown:    []string{"pgdown", "d"},
		Search:      []string{"/"},
		ClearFilter: []string{"ctrl+l"},
		Backspace:   []string{"backspace", "delete"},
		Cancel:      []string{"esc"},
		Confirm:     []string{"enter"},
		Refresh:     []string{"r"},
	}
}

func matches(msg tea.KeyMsg, bindings []string) bool {
	key := strings.ToLower(msg.String())
	for _, b := range bindings {
		if key == strings.ToLower(b) {
			return true
		}
	}
	return false
}
