package core

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MSmaili/hetki/internal/tui/contracts"
)

func (m model) View() tea.View {
	t := m.theme
	lineWidth := m.width
	if lineWidth <= 0 {
		lineWidth = 100
	}
	lineWidth = max(48, lineWidth)

	borderFrameSize := t.appBorder.GetHorizontalFrameSize()
	innerW := max(24, lineWidth-borderFrameSize)

	start := m.offset
	end := m.offset + m.listH
	if end > len(m.rows) {
		end = len(m.rows)
	}

	var contentLines []string

	contentLines = append(contentLines, t.sectionLine.Render(strings.Repeat("─", innerW)))
	contentLines = append(contentLines, truncateToWidth(m.topBar(innerW), innerW))
	contentLines = append(contentLines, t.sectionLine.Render(strings.Repeat("─", innerW)))

	if len(m.rows) == 0 {
		contentLines = append(contentLines, t.meta.Render(truncateWidth(emptyStateText(m), innerW)))
	} else {
		var dataLines int
		for i := start; i < end && dataLines < m.listH; i++ {
			r := m.rows[i]
			if r.Depth == 0 && r.Node.Kind == contracts.NodeKindSession && len(contentLines) > 3 {
				dataLines++
				if dataLines > m.listH {
					break
				}
				contentLines = append(contentLines, t.sectionLine.Render(strings.Repeat("╌", innerW)))
			}
			dataLines++
			if dataLines > m.listH {
				break
			}
			cursor := " "
			if i == m.cursor {
				cursor = "❯"
			}

			active := ""
			if r.Node.Active {
				active = "●"
			}

			selected := i == m.cursor
			line := m.renderRowLine(r, cursor, active, innerW, selected)
			line = m.withSessionCount(r, line, innerW)
			line = m.styleRowLine(r, selected, line)
			contentLines = append(contentLines, truncateToWidth(line, innerW))
		}
	}

	statusText := m.status
	if statusText == "" {
		statusText = strings.ToUpper(m.modeLabel())
	}
	if m.busy {
		statusText += " | busy"
	}
	if m.mode == modeFilter {
		statusText += " | filter=" + m.filter
		if current, total := m.filterMatchPosition(); total > 0 {
			statusText += fmt.Sprintf(" (%d/%d)", current, total)
		}
	}
	if m.err != nil {
		contentLines = append(contentLines, t.err.Render(truncateWidth(statusText, innerW)))
	} else {
		contentLines = append(contentLines, t.status.Render(truncateWidth(statusText, innerW)))
	}

	contentLines = append(contentLines, t.help.Render(truncateWidth(compositeBottomLine(m, innerW), innerW)))

	content := strings.Join(contentLines, "\n")

	rendered := t.appBorder.Width(lineWidth).Render(content)

	var overlayContent string
	if m.mode == modeInput {
		overlayContent = m.renderInputModal(lineWidth)
	} else if m.mode == modeConfirm {
		overlayContent = m.renderConfirmModal(lineWidth)
	} else if m.helpOpen {
		overlayContent = m.renderHelpOverlay(lineWidth)
	}

	if overlayContent != "" {
		totalH := lipgloss.Height(rendered)
		overlayH := lipgloss.Height(overlayContent)
		overlayW := lipgloss.Width(overlayContent)
		x := (lineWidth - overlayW) / 2
		y := (totalH - overlayH) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		bg := lipgloss.NewLayer(rendered)
		fg := lipgloss.NewLayer(overlayContent).X(x).Y(y).Z(1)
		rendered = lipgloss.NewCompositor(bg, fg).Render()
	}

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}

func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	visualW := lipgloss.Width(s)
	if visualW <= width {
		return s
	}
	r := []rune(s)
	accum := 0
	last := 0
	for i := 0; i < len(r); i++ {
		seg := string(r[i])
		segW := lipgloss.Width(seg)
		if accum+segW > width {
			break
		}
		accum += segW
		last = i + 1
	}
	if last >= len(r) {
		return s
	}
	if last <= 0 {
		return ""
	}
	return string(r[:last])
}

func (m model) topBar(width int) string {
	search := strings.TrimSpace(m.filter)
	prompt := "\uf002 " // nerd font: search
	if m.mode == modeFilter {
		return m.theme.searchBox.Render(truncateWidth(prompt+search+"_", width))
	}
	if search != "" {
		return m.theme.searchBox.Render(truncateWidth(prompt+search, width))
	}
	return m.theme.searchBox.Render(prompt)
}

func (m model) lineWithTag(left, tag string, width int) string {
	left = truncateWidth(left, width)
	if strings.TrimSpace(tag) == "" {
		return left
	}
	tag = m.theme.selectedHint.Render(tag)
	tagW := lipgloss.Width(tag)
	if tagW+2 >= width {
		return truncateWidth(left, width)
	}
	leftW := lipgloss.Width(left)
	if leftW+tagW+1 >= width {
		left = truncateWidth(left, width-tagW-1)
		leftW = lipgloss.Width(left)
	}
	padding := width - leftW - tagW
	if padding < 1 {
		padding = 1
	}
	return left + strings.Repeat(" ", padding) + tag
}

func branchGlyph(r row) string {
	if !r.Branch {
		return "  "
	}
	if r.Expanded {
		return "\uf0d7 " // nerd font: fa-caret-down (expanded)
	}
	return "\uf0da " // nerd font: fa-caret-right (collapsed)
}

func (m model) renderRowLine(r row, cursor, active string, width int, selected bool) string {
	left := fmt.Sprintf("%s %s", cursor, m.decoratedLabel(r, selected))
	return m.lineWithTag(left, active, width)
}

func (m model) withSessionCount(r row, line string, width int) string {
	count := sessionWindowCount(r)
	if count == 0 {
		return line
	}
	suffix := fmt.Sprintf("%d", count)
	suffixW := lipgloss.Width(suffix)
	lineW := lipgloss.Width(line)
	if lineW+suffixW+1 >= width {
		return line
	}
	return line + strings.Repeat(" ", width-lineW-suffixW) + m.theme.meta.Render(suffix)
}

func (m model) styleRowLine(r row, selected bool, line string) string {
	t := m.theme
	if selected {
		return t.selectedRow.Render(line)
	}
	if r.Node.Kind == contracts.NodeKindSession {
		return t.sessionRow.Render(line)
	}
	if r.Node.Active {
		return t.activeRow.Render(line)
	}
	if r.Node.Kind == contracts.NodeKindWindow {
		return t.windowRow.Render(line)
	}
	return t.row.Render(line)
}

func sessionIcon() string {
	return "\U000f018d" // nerd font: terminal (material)
}

func windowIcon() string {
	return "\uf489" // nerd font: oct-terminal
}

func nodeIcon(r row) string {
	if r.Node.Kind == contracts.NodeKindSession {
		return sessionIcon()
	}
	return windowIcon()
}

func (m model) decoratedLabel(r row, selected bool) string {
	label := strings.TrimSpace(r.Node.Label)
	if label == "" {
		label = r.Node.ID
	}
	prefix := r.TreePrefix
	branch := branchGlyph(r)
	if !selected {
		prefix = m.theme.rail.Render(prefix)
		branch = m.theme.rail.Render(branch)
	}
	if r.Depth == 0 {
		return fmt.Sprintf("%s %s %s", branch, nodeIcon(r), label)
	}
	return fmt.Sprintf("%s%s %s", prefix, nodeIcon(r), label)
}

func emptyStateText(m model) string {
	if strings.TrimSpace(m.filter) != "" {
		return "no matching sessions or windows | esc cancel filter | ctrl+l clear"
	}
	return "no sessions found"
}

func compositeBottomLine(m model, lineWidth int) string {
	position := "0/0"
	if len(m.rows) > 0 {
		position = fmt.Sprintf("%d/%d", m.cursor+1, len(m.rows))
	}
	ws := workspaceContext(m.snapshot.ContextBars)
	right := "? help"
	if ws == "" {
		return position + "  " + right
	}
	leftW := lipgloss.Width(position)
	rightW := lipgloss.Width(right)
	centerW := lipgloss.Width(ws)
	available := lineWidth - leftW - rightW
	if available < 1 {
		available = 1
	}
	if leftW+centerW+rightW >= lineWidth {
		return position + "  " + ws + "  " + right
	}
	gap := lineWidth - leftW - centerW - rightW
	leftGap := gap / 2
	rightGap := gap - leftGap
	if leftGap < 1 {
		leftGap = 1
	}
	if rightGap < 1 {
		rightGap = 1
	}
	return position + strings.Repeat(" ", leftGap) + ws + strings.Repeat(" ", rightGap) + right
}

func (m model) renderHelpOverlay(lineWidth int) string {
	t := m.theme
	type entry struct{ keys, desc string }
	type section struct {
		title   string
		entries []entry
	}
	sections := []section{
		{"NAVIGATION", []entry{
			{"j, k, ↓, ↑", "move"},
			{"ctrl+n, ctrl+p", "move (vim)"},
			{"u, d, pgup, pgdn", "page up/down"},
			{"g, G", "top / bottom"},
			{"h, l, ←, →", "collapse / expand"},
			{"H, L", "collapse all / expand all"},
		}},
		{"ACTIONS", []entry{
			{"enter, ctrl+y", "select / switch"},
			{"a", "new window"},
			{"s", "new session"},
			{"e", "rename"},
			{"x", "delete"},
			{"r", "refresh"},
		}},
		{"FILTER", []entry{
			{"/", "start filter"},
			{"n, N", "next / prev match"},
			{"ctrl+l", "clear filter"},
		}},
		{"EDIT", []entry{
			{"ctrl+w", "delete word"},
			{"ctrl+u", "clear line"},
		}},
		{"OTHER", []entry{
			{"esc", "cancel"},
			{"q, ctrl+c", "quit"},
			{"?", "toggle help"},
		}},
	}

	// Compute key column width for alignment.
	keyColW := 0
	for _, s := range sections {
		for _, e := range s.entries {
			if w := lipgloss.Width(e.keys); w > keyColW {
				keyColW = w
			}
		}
	}

	var lines []string
	lines = append(lines, t.modalTitle.Render("KEYBINDINGS"))
	for _, s := range sections {
		lines = append(lines, "")
		lines = append(lines, t.meta.Render(s.title))
		for _, e := range s.entries {
			pad := keyColW - lipgloss.Width(e.keys)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, "  "+t.selectedHint.Render(e.keys)+strings.Repeat(" ", pad)+"  "+e.desc)
		}
	}
	lines = append(lines, "")
	lines = append(lines, t.modalHint.Render("? / esc / enter close"))

	return t.helpOverlay.Width(min(76, lineWidth-8)).Render(strings.Join(lines, "\n"))
}

func (m model) renderInputModal(lineWidth int) string {
	t := m.theme
	content := strings.Join([]string{
		t.modalTitle.Render(m.input.Title),
		m.input.Prompt,
		"> " + m.input.Value + "_",
		t.modalHint.Render("enter submit | esc cancel"),
	}, "\n")
	return t.modal.Width(min(72, lineWidth-8)).Render(content)
}

func (m model) renderConfirmModal(lineWidth int) string {
	t := m.theme
	content := strings.Join([]string{
		t.modalTitle.Render(m.confirm.Title),
		m.confirm.Body,
		t.modalHint.Render("enter/y confirm | esc/n cancel"),
	}, "\n")
	return t.modal.Width(min(72, lineWidth-8)).Render(content)
}
