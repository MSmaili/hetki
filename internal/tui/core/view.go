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
	var rowLines []string

	contentLines = append(contentLines, t.sectionLine.Render(strings.Repeat("─", innerW)))
	contentLines = append(contentLines, truncateToWidth(m.topBar(innerW), innerW))
	contentLines = append(contentLines, t.sectionLine.Render(strings.Repeat("─", innerW)))

	if len(m.rows) == 0 {
		rowLines = append(rowLines, t.meta.Render(truncateWidth(emptyStateText(m), innerW)))
	} else {
		var dataLines int
		for i := start; i < end && dataLines < m.listH; i++ {
			r := m.rows[i]
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
			line := m.renderRowLine(r, cursor, active, selected)
			line = truncateWidth(line, innerW)
			line = m.styleRowLine(r, selected, line, innerW)
			rowLines = append(rowLines, line)
		}
	}

	statusText := m.status
	if statusText == "" {
		statusText = strings.ToUpper(m.modeLabel())
	}
	if m.busy {
		statusText += " · busy"
	}

	var statusStyle lipgloss.Style
	if m.err != nil {
		statusStyle = t.err
	} else {
		statusStyle = t.status
	}
	bottomBar := m.renderBottomBar(innerW, statusText, statusStyle)

	// Compose the view as three stacked blocks: header, middle (row list),
	// and bottom bar. The middle region is padded to fill the available
	// vertical space so the bottom bar sticks to the bottom.
	header := strings.Join(contentLines, "\n")
	middle := strings.Join(rowLines, "\n")
	borderFrameV := t.appBorder.GetVerticalFrameSize()
	middleH := m.height - borderFrameV - lipgloss.Height(header) - lipgloss.Height(bottomBar)
	if middleH < 1 {
		middleH = 1
	}
	middle = lipgloss.PlaceVertical(middleH, lipgloss.Top, middle)

	content := lipgloss.JoinVertical(lipgloss.Left, header, middle, bottomBar)

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

func branchGlyph(r row) string {
	if !r.Branch {
		return "  "
	}
	if r.Expanded {
		return "\uf0d7 " // nerd font: fa-caret-down (expanded)
	}
	return "\uf0da " // nerd font: fa-caret-right (collapsed)
}

func (m model) renderRowLine(r row, cursor, active string, selected bool) string {
	marker := "  "
	if strings.TrimSpace(active) != "" {
		marker = active + " "
	}
	label := m.decoratedLabel(r, selected)
	if count := sessionWindowCount(r); count > 0 {
		label += " " + m.theme.meta.Render(fmt.Sprintf("(%d)", count))
	}
	return fmt.Sprintf("%s %s%s", cursor, marker, label)
}

func (m model) styleRowLine(r row, selected bool, line string, width int) string {
	t := m.theme
	if selected {
		return t.selectedRow.Width(width).Render(line)
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
	return "\ueb14" // nerd font: codicon-window
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
	return fmt.Sprintf("  %s%s %s", prefix, nodeIcon(r), label)
}

func emptyStateText(m model) string {
	if strings.TrimSpace(m.filter) != "" {
		return "no matching sessions or windows | esc cancel filter | ctrl+l clear"
	}
	return "no sessions found"
}

func (m model) renderBottomBar(width int, status string, statusStyle lipgloss.Style) string {
	t := m.theme

	// Right side: position/match counter + help hint.
	right := "? help"
	if len(m.rows) > 0 {
		right = fmt.Sprintf("%d/%d · %s", m.cursor+1, len(m.rows), right)
	}
	if m.mode == modeFilter {
		if current, total := m.filterMatchPosition(); total > 0 {
			right = fmt.Sprintf("match %d/%d · %s", current, total, right)
		}
	}

	// Center: workspace label if present.
	center := workspaceContext(m.snapshot.ContextBars)

	leftStyled := statusStyle.Render(status)
	rightStyled := t.help.Render(right)
	centerStyled := t.meta.Render(center)

	leftW := lipgloss.Width(leftStyled)
	rightW := lipgloss.Width(rightStyled)
	centerW := lipgloss.Width(centerStyled)

	// If everything doesn't fit, drop the center.
	if leftW+centerW+rightW+4 > width {
		centerStyled = ""
		centerW = 0
	}
	// If still doesn't fit, drop the center and truncate the status.
	if leftW+rightW+2 > width {
		leftStyled = statusStyle.Render(truncateWidth(status, width-rightW-2))
		leftW = lipgloss.Width(leftStyled)
	}

	if centerW == 0 {
		gap := width - leftW - rightW
		if gap < 1 {
			gap = 1
		}
		return leftStyled + strings.Repeat(" ", gap) + rightStyled
	}

	gap := width - leftW - centerW - rightW
	leftGap := gap / 2
	rightGap := gap - leftGap
	if leftGap < 1 {
		leftGap = 1
	}
	if rightGap < 1 {
		rightGap = 1
	}
	return leftStyled + strings.Repeat(" ", leftGap) + centerStyled + strings.Repeat(" ", rightGap) + rightStyled
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
