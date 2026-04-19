package core

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Quit          []string
	Up            []string
	Down          []string
	Top           []string
	Bottom        []string
	PageUp        []string
	PageDown      []string
	Search        []string
	Help          []string
	NextMatch     []string
	PrevMatch     []string
	ClearFilter   []string
	CreateSession []string
	CreateWindow  []string
	Rename        []string
	Delete        []string
	Expand        []string
	Collapse      []string
	ExpandAll     []string
	CollapseAll   []string
	Backspace     []string
	DeleteWord    []string
	DeleteToStart []string
	Cancel        []string
	Confirm       []string
	Refresh       []string
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:          []string{"q", "ctrl+c"},
		Up:            []string{"up", "k"},
		Down:          []string{"down", "j"},
		Top:           []string{"g"},
		Bottom:        []string{"G"},
		PageUp:        []string{"pgup", "u"},
		PageDown:      []string{"pgdown", "d"},
		Search:        []string{"/"},
		Help:          []string{"?"},
		NextMatch:     []string{"n"},
		PrevMatch:     []string{"N"},
		ClearFilter:   []string{"ctrl+l"},
		CreateSession: []string{"s"},
		CreateWindow:  []string{"a"},
		Rename:        []string{"e"},
		Delete:        []string{"x"},
		Expand:        []string{"right", "l"},
		Collapse:      []string{"left", "h"},
		ExpandAll:     []string{"L"},
		CollapseAll:   []string{"H"},
		Backspace:     []string{"backspace", "delete"},
		DeleteWord:    []string{"ctrl+w"},
		DeleteToStart: []string{"ctrl+u"},
		Cancel:        []string{"esc"},
		Confirm:       []string{"enter"},
		Refresh:       []string{"r"},
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

func exactMatches(msg tea.KeyMsg, bindings []string) bool {
	key := msg.String()
	for _, b := range bindings {
		if key == b {
			return true
		}
	}
	return false
}
