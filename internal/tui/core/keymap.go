package core

import (
	"charm.land/bubbles/v2/key"
)

type KeyMap struct {
	Quit          key.Binding
	Up            key.Binding
	Down          key.Binding
	Top           key.Binding
	Bottom        key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	Search        key.Binding
	Help          key.Binding
	NextMatch     key.Binding
	PrevMatch     key.Binding
	ClearFilter   key.Binding
	CreateSession key.Binding
	CreateWindow  key.Binding
	Rename        key.Binding
	Delete        key.Binding
	Expand        key.Binding
	Collapse      key.Binding
	ExpandAll     key.Binding
	CollapseAll   key.Binding
	Backspace     key.Binding
	DeleteWord    key.Binding
	DeleteToStart key.Binding
	Cancel        key.Binding
	Confirm       key.Binding
	Refresh       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Up:            key.NewBinding(key.WithKeys("up", "k", "ctrl+p"), key.WithHelp("k/↑/ctrl+p", "up")),
		Down:          key.NewBinding(key.WithKeys("down", "j", "ctrl+n"), key.WithHelp("j/↓/ctrl+n", "down")),
		Top:           key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:        key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		PageUp:        key.NewBinding(key.WithKeys("pgup", "u"), key.WithHelp("u", "page up")),
		PageDown:      key.NewBinding(key.WithKeys("pgdown", "d"), key.WithHelp("d", "page down")),
		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		NextMatch:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		PrevMatch:     key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		ClearFilter:   key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear filter")),
		CreateSession: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "new session")),
		CreateWindow:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new window")),
		Rename:        key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "rename")),
		Delete:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		Expand:        key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l", "expand")),
		Collapse:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h", "collapse")),
		ExpandAll:     key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "expand all")),
		CollapseAll:   key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "collapse all")),
		Backspace:     key.NewBinding(key.WithKeys("backspace", "delete")),
		DeleteWord:    key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "delete word")),
		DeleteToStart: key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear line")),
		Cancel:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Confirm:       key.NewBinding(key.WithKeys("enter", "ctrl+y"), key.WithHelp("enter/ctrl+y", "confirm")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}
