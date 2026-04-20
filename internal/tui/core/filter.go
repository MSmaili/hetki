package core

import "strings"

func (m *model) applyFilter() {
	if strings.TrimSpace(m.filter) == "" {
		m.rows = flatten(m.snapshot.Nodes, m.expanded, false)
		m.cursor = clampCursor(m.cursor, len(m.rows))
		return
	}

	query := strings.ToLower(strings.TrimSpace(m.filter))
	if query == "" {
		m.rows = flatten(m.snapshot.Nodes, m.expanded, false)
		m.cursor = clampCursor(m.cursor, len(m.rows))
		return
	}

	allRows := flatten(m.snapshot.Nodes, m.expanded, true)
	parentByID := make(map[string]string, len(allRows))
	keep := make(map[string]bool, len(allRows))

	for _, r := range allRows {
		parentByID[r.Node.ID] = r.Node.ParentID
		if strings.Contains(strings.ToLower(r.Node.Label), query) {
			keep[r.Node.ID] = true
		}
	}

	for id := range keep {
		for parent := parentByID[id]; parent != ""; parent = parentByID[parent] {
			keep[parent] = true
		}
	}

	filtered := make([]row, 0, len(allRows))
	for _, r := range allRows {
		if keep[r.Node.ID] {
			filtered = append(filtered, r)
		}
	}

	m.rows = filtered
	if matchIndices := m.matchIndices(); len(matchIndices) > 0 {
		m.cursor = matchIndices[0]
	} else {
		m.cursor = clampCursor(m.cursor, len(m.rows))
	}
}

func (m model) filterQuery() string {
	return strings.ToLower(strings.TrimSpace(m.filter))
}

func (m model) matchIndices() []int {
	query := m.filterQuery()
	if query == "" {
		return nil
	}
	indices := make([]int, 0)
	for i, r := range m.rows {
		if strings.Contains(strings.ToLower(strings.TrimSpace(r.Node.Label)), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *model) jumpMatch(forward bool) bool {
	indices := m.matchIndices()
	if len(indices) == 0 {
		return false
	}
	if len(indices) == 1 {
		m.cursor = indices[0]
		*m = m.reflow()
		return true
	}
	currentPos := 0
	for i, idx := range indices {
		if idx == m.cursor {
			currentPos = i
			break
		}
		if idx > m.cursor {
			currentPos = i
			if !forward {
				currentPos = i - 1
			}
			break
		}
		currentPos = i
	}
	if forward {
		currentPos = (currentPos + 1) % len(indices)
	} else {
		currentPos = (currentPos - 1 + len(indices)) % len(indices)
	}
	m.cursor = indices[currentPos]
	*m = m.reflow()
	return true
}

func (m model) filterMatchPosition() (int, int) {
	indices := m.matchIndices()
	if len(indices) == 0 {
		return 0, 0
	}
	for i, idx := range indices {
		if idx == m.cursor {
			return i + 1, len(indices)
		}
	}
	return 1, len(indices)
}
