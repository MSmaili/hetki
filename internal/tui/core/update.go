package core

import (
	"context"

	"github.com/MSmaili/hetki/internal/tui/contracts"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.reflow()
		return m, nil
	case actionResultMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			m.status = msg.err.Error()
			return m, nil
		}
		m.err = nil
		if msg.result.Message != "" {
			m.status = msg.result.Message
		}
		if msg.result.Snapshot != nil {
			selectedID := ""
			if selected, ok := m.selectedRow(); ok {
				selectedID = selected.Node.ID
			}
			m.snapshot = *msg.result.Snapshot
			m.allRows = flatten(m.snapshot.Nodes)
			m.applyFilter()
			m = m.reflow()
			if selectedID != "" {
				for i := range m.rows {
					if m.rows[i].Node.ID == selectedID {
						m.cursor = i
						break
					}
				}
			}
			m = m.reflow()
		}
		return m, nil
	case tea.KeyMsg:
		if matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if m.busy {
			return m, nil
		}

		if m.mode == modeFilter {
			switch {
			case matches(msg, m.keys.Cancel):
				m.mode = modeBrowse
				m.status = "filter canceled"
				return m, nil
			case matches(msg, m.keys.Confirm):
				m.mode = modeBrowse
				m.status = "filter applied"
				return m, nil
			case matches(msg, m.keys.Backspace):
				if len(m.filter) > 0 {
					r := []rune(m.filter)
					m.filter = string(r[:len(r)-1])
					m.applyFilter()
					m = m.reflow()
				}
				return m, nil
			case matches(msg, m.keys.ClearFilter):
				m.filter = ""
				m.applyFilter()
				m = m.reflow()
				m.mode = modeBrowse
				m.status = "filter cleared"
				return m, nil
			}

			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
				m.applyFilter()
				m = m.reflow()
				return m, nil
			}

			return m, nil
		}

		switch {
		case matches(msg, m.keys.Up):
			m.cursor = clampCursor(m.cursor-1, len(m.rows))
			m = m.reflow()
			return m, nil
		case matches(msg, m.keys.Down):
			m.cursor = clampCursor(m.cursor+1, len(m.rows))
			m = m.reflow()
			return m, nil
		case matches(msg, m.keys.PageUp):
			m.cursor = clampCursor(m.cursor-m.listH, len(m.rows))
			m = m.reflow()
			return m, nil
		case matches(msg, m.keys.PageDown):
			m.cursor = clampCursor(m.cursor+m.listH, len(m.rows))
			m = m.reflow()
			return m, nil
		case matches(msg, m.keys.Search):
			m.mode = modeFilter
			m.status = "type to filter, enter apply, esc cancel"
			return m, nil
		case matches(msg, m.keys.ClearFilter):
			m.filter = ""
			m.applyFilter()
			m = m.reflow()
			m.status = "filter cleared"
			return m, nil
		case matches(msg, m.keys.Refresh):
			if !m.hasCapability(contracts.CapabilityRefresh) {
				m.status = "refresh is not available"
				return m, nil
			}
			m.busy = true
			m.status = "refreshing..."
			return m, runIntent(m.dispatch, contracts.Intent{Type: contracts.IntentRefresh})
		case matches(msg, m.keys.Confirm):
			if !m.hasCapability(contracts.CapabilitySwitch) {
				m.status = "switch is not available"
				return m, nil
			}
			selected, ok := m.selectedRow()
			if !ok {
				m.status = "no selection"
				return m, nil
			}
			if selected.Node.Target == "" {
				m.status = "selected item is not actionable"
				return m, nil
			}
			m.busy = true
			m.status = "switching..."
			return m, runIntent(m.dispatch, contracts.Intent{
				Type:   contracts.IntentSwitch,
				Target: selected.Node.Target,
			})
		}
	}

	return m, nil
}

func runIntent(dispatch DispatchFunc, intent contracts.Intent) tea.Cmd {
	if dispatch == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := dispatch(context.Background(), intent)
		return actionResultMsg{result: result, err: err}
	}
}
