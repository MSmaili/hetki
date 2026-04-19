package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	metaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	searchBoxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	rowStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sessionRowStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("224"))
	windowRowStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	activeRowStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("181"))
	selectedRowStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60"))
	selectedHintStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("181"))
	railStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("210")).Bold(true)
	keyBarStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("110"))
	sectionLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	listBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("110")).
			Padding(0, 1)
	modalTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("110"))
	modalHintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

func (m model) View() string {
	m = m.reflow()
	lineWidth := m.width
	if lineWidth <= 0 {
		lineWidth = 100
	}
	lineWidth = max(48, lineWidth)

	var b strings.Builder

	b.WriteString(topBar(m, lineWidth))
	b.WriteString("\n")
	if context := workspaceContext(m.snapshot.ContextBars); context != "" {
		b.WriteString(metaStyle.Render(truncateWidth(context, lineWidth)))
		b.WriteString("\n")
		b.WriteString(sectionLineStyle.Render(strings.Repeat("─", lineWidth)))
		b.WriteString("\n")
	}

	start := m.offset
	end := m.offset + m.listH
	if end > len(m.rows) {
		end = len(m.rows)
	}

	innerW := max(24, lineWidth-6)
	lines := make([]string, 0, m.listH)

	if len(m.rows) == 0 {
		lines = append(lines, metaStyle.Render(truncateWidth(emptyStateText(m), innerW)))
		for i := 1; i < m.listH; i++ {
			lines = append(lines, "")
		}
	} else {
		for i := start; i < end; i++ {
			r := m.rows[i]
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			active := ""
			if r.Node.Active {
				active = "●"
			}

			selected := i == m.cursor
			line := renderRowLine(r, cursor, active, innerW, selected)
			line = withSessionCount(r, line, innerW)
			line = styleRowLine(r, selected, line)
			lines = append(lines, line)
		}
		for i := end; i < start+m.listH; i++ {
			lines = append(lines, "")
		}
	}

	b.WriteString(listBoxStyle.Width(lineWidth - 2).Render(strings.Join(lines, "\n")))
	b.WriteString("\n")

	statusText := strings.ToUpper(m.modeLabel()) + " | " + m.status
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
		b.WriteString(errorStyle.Render(truncateWidth(statusText, lineWidth)))
	} else {
		b.WriteString(statusStyle.Render(truncateWidth(statusText, lineWidth)))
	}
	b.WriteString("\n")

	b.WriteString(helpStyle.Render(truncateWidth(statusLine(m), lineWidth)))
	b.WriteString("\n")
	b.WriteString(keyBarStyle.Render(truncateWidth(contextualActions(m), lineWidth)))
	b.WriteString("\n")

	if m.mode == modeInput {
		b.WriteString(renderInputModal(m, lineWidth))
		b.WriteString("\n")
	}
	if m.mode == modeConfirm {
		b.WriteString(renderConfirmModal(m, lineWidth))
		b.WriteString("\n")
	}
	if m.helpOpen {
		b.WriteString(renderHelpOverlay(lineWidth))
		b.WriteString("\n")
	}

	return b.String()
}

func topBar(m model, width int) string {
	search := strings.TrimSpace(m.filter)
	if m.mode == modeFilter {
		return searchBoxStyle.Render(truncateWidth("> "+search+"_", width))
	}
	if search != "" {
		return searchBoxStyle.Render(truncateWidth("> "+search, width))
	}
	return searchBoxStyle.Render(">")
}

func lineWithTag(left, tag string, width int) string {
	left = truncateWidth(left, width)
	if strings.TrimSpace(tag) == "" {
		return left
	}
	tag = selectedHintStyle.Render(tag)
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
		return " "
	}
	return " "
}

func renderRowLine(r row, cursor, active string, width int, selected bool) string {
	left := fmt.Sprintf("%s %s", cursor, decoratedLabel(r, selected))
	return lineWithTag(left, active, width)
}

func withSessionCount(r row, line string, width int) string {
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
	return line + strings.Repeat(" ", width-lineW-suffixW) + metaStyle.Render(suffix)
}

func styleRowLine(r row, selected bool, line string) string {
	if selected {
		return selectedRowStyle.Render(line)
	}
	if r.Node.Kind == "session" {
		return sessionRowStyle.Render(line)
	}
	if r.Node.Active {
		return activeRowStyle.Render(line)
	}
	if r.Node.Kind == "window" {
		return windowRowStyle.Render(line)
	}
	return rowStyle.Render(line)
}

func sessionIcon() string {
	return "󰆍"
}

func windowIcon() string {
	return ""
}

func nodeIcon(r row) string {
	if r.Node.Kind == "session" {
		return sessionIcon()
	}
	return windowIcon()
}

func decoratedLabel(r row, selected bool) string {
	label := strings.TrimSpace(r.Node.Label)
	if label == "" {
		label = r.Node.ID
	}
	prefix := r.TreePrefix
	branch := branchGlyph(r)
	if !selected {
		prefix = railStyle.Render(prefix)
		branch = railStyle.Render(branch)
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

func contextualActions(m model) string {
	selected, ok := m.selectedRow()
	if !ok {
		return "/ filter  ? help  q exit"
	}
	switch selected.Node.Kind {
	case "session":
		return "enter switch  a window  e rename  x delete  ? help"
	case "window":
		return "enter switch  e rename  x delete  ? help"
	default:
		return "e rename  x delete  ? help"
	}
}

func statusLine(m model) string {
	status := m.status
	if status == "" {
		status = "ready"
	}
	if m.mode == modeFilter {
		if current, total := m.filterMatchPosition(); total > 0 {
			status += fmt.Sprintf(" (%d/%d)", current, total)
		}
	}
	position := "0/0"
	if len(m.rows) > 0 {
		position = fmt.Sprintf("%d/%d", m.cursor+1, len(m.rows))
	}
	return status + " | " + position
}

func renderHelpOverlay(lineWidth int) string {
	content := strings.Join([]string{
		modalTitleStyle.Render("KEYBINDINGS"),
		"j/k move   u/d page   g/G top/bottom",
		"h/l fold   H/L collapse-all/expand-all",
		"/ filter   n/N next/prev match",
		"a new-window   s new-session   e rename   x delete",
		"enter switch   r refresh   ctrl+w delete-word   ctrl+u clear",
		modalHintStyle.Render("? / esc / enter close"),
	}, "\n")
	return centerBlock(helpOverlayStyle.Width(minInt(76, lineWidth-8)).Render(content), lineWidth)
}

func renderInputModal(m model, lineWidth int) string {
	content := strings.Join([]string{
		modalTitleStyle.Render(m.input.Title),
		m.input.Prompt,
		"> " + m.input.Value + "_",
		modalHintStyle.Render("enter submit | esc cancel"),
	}, "\n")
	return centerBlock(modalStyle.Width(minInt(72, lineWidth-8)).Render(content), lineWidth)
}

func renderConfirmModal(m model, lineWidth int) string {
	content := strings.Join([]string{
		modalTitleStyle.Render(m.confirm.Title),
		m.confirm.Body,
		modalHintStyle.Render("enter/y confirm | esc/n cancel"),
	}, "\n")
	return centerBlock(modalStyle.Width(minInt(72, lineWidth-8)).Render(content), lineWidth)
}

func centerBlock(block string, width int) string {
	blockW := lipgloss.Width(block)
	if blockW >= width {
		return block
	}
	padding := (width - blockW) / 2
	if padding < 0 {
		padding = 0
	}
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", padding) + line
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
