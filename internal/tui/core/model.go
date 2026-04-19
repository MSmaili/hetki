package core

import (
	"context"
	"sort"
	"strings"

	"github.com/MSmaili/hetki/internal/tui/contracts"
	tea "github.com/charmbracelet/bubbletea"
)

type DispatchFunc func(context.Context, contracts.Intent) (contracts.ActionResult, error)

func Run(initial contracts.Snapshot, dispatch DispatchFunc) error {
	p := tea.NewProgram(newModel(initial, dispatch), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type row struct {
	Node  contracts.Node
	Depth int
}

type actionResultMsg struct {
	result contracts.ActionResult
	err    error
}

type uiMode string

const (
	modeBrowse uiMode = "browse"
	modeFilter uiMode = "filter"
)

type model struct {
	snapshot contracts.Snapshot
	allRows  []row
	rows     []row
	cursor   int
	offset   int
	listH    int
	mode     uiMode
	filter   string
	status   string
	err      error
	busy     bool

	width  int
	height int

	dispatch DispatchFunc
	keys     KeyMap
}

func newModel(snapshot contracts.Snapshot, dispatch DispatchFunc) model {
	m := model{
		snapshot: snapshot,
		dispatch: dispatch,
		keys:     DefaultKeyMap(),
		mode:     modeBrowse,
		status:   "ready",
	}
	m.allRows = flatten(snapshot.Nodes)
	m.applyFilter()
	m.cursor = clampCursor(0, len(m.rows))
	return m.reflow()
}

func clampCursor(cursor, size int) int {
	if size == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= size {
		return size - 1
	}
	return cursor
}

func flatten(nodes []contracts.Node) []row {
	out := make([]row, 0)
	flattenAtDepth(nodes, 0, &out)
	return out
}

func flattenAtDepth(nodes []contracts.Node, depth int, out *[]row) {
	for _, n := range nodes {
		*out = append(*out, row{Node: n, Depth: depth})
		if len(n.Children) > 0 {
			flattenAtDepth(n.Children, depth+1, out)
		}
	}
}

func (m *model) applyFilter() {
	if strings.TrimSpace(m.filter) == "" {
		m.rows = cloneRows(m.allRows)
		m.cursor = clampCursor(m.cursor, len(m.rows))
		return
	}

	query := strings.ToLower(strings.TrimSpace(m.filter))
	if query == "" {
		m.rows = cloneRows(m.allRows)
		m.cursor = clampCursor(m.cursor, len(m.rows))
		return
	}

	rowsByID := make(map[string]row, len(m.allRows))
	parentByID := make(map[string]string, len(m.allRows))
	keep := make(map[string]bool, len(m.allRows))

	for _, r := range m.allRows {
		rowsByID[r.Node.ID] = r
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

	filtered := make([]row, 0, len(m.allRows))
	for _, r := range m.allRows {
		if keep[r.Node.ID] {
			filtered = append(filtered, r)
		}
	}

	m.rows = filtered
	m.cursor = clampCursor(m.cursor, len(m.rows))
}

func cloneRows(in []row) []row {
	if len(in) == 0 {
		return nil
	}
	out := make([]row, len(in))
	copy(out, in)
	return out
}

func (m model) reflow() model {
	m.cursor = clampCursor(m.cursor, len(m.rows))
	m.listH = m.availableListHeight()
	m.offset = clampOffset(m.offset, len(m.rows), m.listH)
	m.offset = ensureCursorVisible(m.offset, m.cursor, len(m.rows), m.listH)
	return m
}

func (m model) availableListHeight() int {
	height := m.height
	if height <= 0 {
		height = 24
	}
	list := height - m.reservedLines()
	if list < 3 {
		return 3
	}
	return list
}

func (m model) reservedLines() int {
	// Header/title/meta/filter (4) plus status/help/footer (4).
	return 8
}

func clampOffset(offset, total, height int) int {
	if total <= 0 || height <= 0 {
		return 0
	}
	maxOffset := total - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func ensureCursorVisible(offset, cursor, total, height int) int {
	if total <= 0 || height <= 0 {
		return 0
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+height {
		offset = cursor - height + 1
	}
	return clampOffset(offset, total, height)
}

func (m model) selectedRow() (row, bool) {
	if len(m.rows) == 0 {
		return row{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return row{}, false
	}
	return m.rows[m.cursor], true
}

func (m model) hasCapability(c contracts.Capability) bool {
	if m.snapshot.Capabilities == nil {
		return false
	}
	return m.snapshot.Capabilities[c]
}

func contextLines(ctx map[string]string) []string {
	if len(ctx) == 0 {
		return nil
	}
	keys := make([]string, 0, len(ctx))
	for k := range ctx {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+": "+ctx[k])
	}
	return lines
}

func rowLabel(r row) string {
	indent := strings.Repeat("  ", r.Depth)
	if r.Depth == 0 {
		return r.Node.Label
	}
	return indent + "- " + r.Node.Label
}

func truncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}

func (m model) modeLabel() string {
	if m.mode == modeFilter {
		return "filter"
	}
	return "browse"
}
