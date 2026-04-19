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
	Node       contracts.Node
	Depth      int
	TreePrefix string
	Expanded   bool
	Branch     bool
}

type actionResultMsg struct {
	result contracts.ActionResult
	err    error
}

type uiMode string

const (
	modeBrowse  uiMode = "browse"
	modeFilter  uiMode = "filter"
	modeInput   uiMode = "input"
	modeConfirm uiMode = "confirm"
)

type inputState struct {
	Title        string
	Prompt       string
	IntentType   contracts.IntentType
	Target       string
	Payload      map[string]string
	Value        string
	SubmitStatus string
}

type confirmState struct {
	Title        string
	Body         string
	Intent       contracts.Intent
	SubmitStatus string
}

type model struct {
	snapshot contracts.Snapshot
	rows     []row
	cursor   int
	offset   int
	listH    int
	mode     uiMode
	filter   string
	status   string
	err      error
	busy     bool
	input    inputState
	confirm  confirmState
	expanded map[string]bool
	pending  *contracts.Intent
	helpOpen bool

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
		expanded: defaultExpanded(snapshot.Nodes, snapshot.ActiveNodeID),
	}
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

func flatten(nodes []contracts.Node, expanded map[string]bool, includeAll bool) []row {
	out := make([]row, 0)
	flattenAtDepth(nodes, nil, expanded, includeAll, &out)
	return out
}

func flattenAtDepth(nodes []contracts.Node, ancestors []bool, expanded map[string]bool, includeAll bool, out *[]row) {
	for i, n := range nodes {
		hasNext := i < len(nodes)-1
		depth := len(ancestors)
		isExpanded := includeAll || expanded[n.ID]
		*out = append(*out, row{
			Node:       n,
			Depth:      depth,
			TreePrefix: treePrefix(ancestors, hasNext),
			Expanded:   isExpanded,
			Branch:     len(n.Children) > 0,
		})
		if len(n.Children) > 0 {
			nextAncestors := append(append([]bool(nil), ancestors...), hasNext)
			if isExpanded {
				flattenAtDepth(n.Children, nextAncestors, expanded, includeAll, out)
			}
		}
	}
}

func defaultExpanded(nodes []contracts.Node, activeNodeID string) map[string]bool {
	expanded := make(map[string]bool)
	markAllExpanded(nodes, expanded)
	if activeNodeID != "" {
		markActivePathExpanded(nodes, activeNodeID, expanded)
	}
	return expanded
}

func markActivePathExpanded(nodes []contracts.Node, activeNodeID string, expanded map[string]bool) bool {
	for _, n := range nodes {
		if n.ID == activeNodeID {
			expanded[n.ID] = true
			return true
		}
		if markActivePathExpanded(n.Children, activeNodeID, expanded) {
			expanded[n.ID] = true
			return true
		}
	}
	return false
}

func treePrefix(ancestors []bool, hasNext bool) string {
	if len(ancestors) == 0 {
		return ""
	}

	var b strings.Builder
	for _, ancestorHasNext := range ancestors[:len(ancestors)-1] {
		if ancestorHasNext {
			b.WriteString("   |  ")
		} else {
			b.WriteString("      ")
		}
	}
	b.WriteString("   |  ")
	return b.String()
}

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
	reserved := 9
	if m.mode == modeInput || m.mode == modeConfirm {
		reserved += 5
	}
	return reserved
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

func (m *model) toggleCurrentRow(expand bool) bool {
	selected, ok := m.selectedRow()
	if !ok || !selected.Branch || strings.TrimSpace(m.filter) != "" {
		return false
	}
	if m.expanded == nil {
		m.expanded = make(map[string]bool)
	}
	m.expanded[selected.Node.ID] = expand
	m.applyFilter()
	*m = m.reflow()
	return true
}

func (m *model) collapseAll() bool {
	if strings.TrimSpace(m.filter) != "" {
		return false
	}
	m.expanded = make(map[string]bool)
	m.applyFilter()
	*m = m.reflow()
	return true
}

func (m *model) expandAll() bool {
	if strings.TrimSpace(m.filter) != "" {
		return false
	}
	if m.expanded == nil {
		m.expanded = make(map[string]bool)
	}
	markAllExpanded(m.snapshot.Nodes, m.expanded)
	m.applyFilter()
	*m = m.reflow()
	return true
}

func markAllExpanded(nodes []contracts.Node, expanded map[string]bool) {
	for _, n := range nodes {
		if len(n.Children) > 0 {
			expanded[n.ID] = true
			markAllExpanded(n.Children, expanded)
		}
	}
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
	label := strings.TrimSpace(r.Node.Label)
	if label == "" {
		label = r.Node.ID
	}
	if r.Depth == 0 {
		return label
	}
	return r.TreePrefix + label
}

func workspaceContext(ctx map[string]string) string {
	if ctx == nil {
		return ""
	}
	workspace := strings.TrimSpace(ctx["workspace"])
	if workspace == "" {
		return ""
	}
	return "WORKSPACE: " + workspace
}

func sessionWindowCount(r row) int {
	if r.Node.Kind != contracts.NodeKindSession {
		return 0
	}
	return len(r.Node.Children)
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
	switch m.mode {
	case modeFilter:
		return "filter"
	case modeInput:
		return "input"
	case modeConfirm:
		return "confirm"
	default:
		return "browse"
	}
}

func cloneIntent(intent contracts.Intent) *contracts.Intent {
	cloned := contracts.Intent{
		Type:    intent.Type,
		Target:  intent.Target,
		Payload: clonePayloadMap(intent.Payload),
	}
	return &cloned
}

func clonePayloadMap(payload map[string]string) map[string]string {
	if payload == nil {
		return nil
	}
	cloned := make(map[string]string, len(payload))
	for k, v := range payload {
		cloned[k] = v
	}
	return cloned
}

func findRowIndexByID(rows []row, nodeID string) int {
	if nodeID == "" {
		return -1
	}
	for i := range rows {
		if rows[i].Node.ID == nodeID {
			return i
		}
	}
	return -1
}

func preferredSelectionID(snapshot contracts.Snapshot, intent *contracts.Intent, previousID string) string {
	if intent == nil {
		return previousID
	}
	switch intent.Type {
	case contracts.IntentCreateSession, contracts.IntentRenameSession:
		name := strings.TrimSpace(intent.Payload["name"])
		if name != "" {
			return "session:" + name
		}
	case contracts.IntentCreateWindow:
		session := strings.TrimSpace(intent.Payload["session"])
		if session == "" {
			session = sessionFromNodeTarget(intent.Target)
		}
		name := strings.TrimSpace(intent.Payload["name"])
		if session != "" && name != "" {
			if id := findWindowNodeIDByName(snapshot.Nodes, session, name); id != "" {
				return id
			}
		}
	case contracts.IntentRenameWindow:
		return nodeIDFromTarget(intent.Target)
	case contracts.IntentDeleteWindow:
		return "session:" + sessionFromNodeTarget(intent.Target)
	}
	return previousID
}

func nodeIDFromTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if session, window, ok := sessionWindowFromNodeTarget(target); ok {
		return "window:" + session + ":" + window
	}
	return "session:" + target
}

func sessionWindowFromNodeTarget(target string) (string, string, bool) {
	session, rest, hasWindow := strings.Cut(strings.TrimSpace(target), ":")
	if !hasWindow || session == "" || rest == "" {
		return "", "", false
	}
	window, _, _ := strings.Cut(rest, ".")
	window = strings.TrimSpace(window)
	if window == "" {
		return "", "", false
	}
	return strings.TrimSpace(session), window, true
}

func findWindowNodeIDByName(nodes []contracts.Node, sessionName, windowName string) string {
	for _, session := range nodes {
		if session.Kind != contracts.NodeKindSession || session.Label != sessionName {
			continue
		}
		for _, window := range session.Children {
			if windowDisplayName(window.Label) == windowName {
				return window.ID
			}
		}
	}
	return ""
}

func windowDisplayName(label string) string {
	parts := strings.Fields(strings.TrimSpace(label))
	if len(parts) <= 1 {
		return strings.TrimSpace(label)
	}
	return strings.Join(parts[1:], " ")
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
