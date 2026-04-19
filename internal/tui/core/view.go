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
	sectionLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	appBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("110")).
			Padding(0, 2)

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

	innerW := max(24, lineWidth-appBorderWidth())

	start := m.offset
	end := m.offset + m.listH
	if end > len(m.rows) {
		end = len(m.rows)
	}

	var contentLines []string

	contentLines = append(contentLines, sectionLineStyle.Render(strings.Repeat("─", innerW)))
	contentLines = append(contentLines, truncateToWidth(topBar(m, innerW), innerW))
	contentLines = append(contentLines, sectionLineStyle.Render(strings.Repeat("─", innerW)))

	if len(m.rows) == 0 {
		contentLines = append(contentLines, metaStyle.Render(truncateWidth(emptyStateText(m), innerW)))
	} else {
		var dataLines int
		for i := start; i < end && dataLines < m.listH; i++ {
			r := m.rows[i]
			if r.Depth == 0 && r.Node.Kind == "session" && len(contentLines) > 3 {
				dataLines++
				if dataLines > m.listH {
					break
				}
				contentLines = append(contentLines, sectionLineStyle.Render(strings.Repeat("╌", innerW)))
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
			line := renderRowLine(r, cursor, active, innerW, selected)
			line = withSessionCount(r, line, innerW)
			line = styleRowLine(r, selected, line)
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
		contentLines = append(contentLines, errorStyle.Render(truncateWidth(statusText, innerW)))
	} else {
		contentLines = append(contentLines, statusStyle.Render(truncateWidth(statusText, innerW)))
	}

	contentLines = append(contentLines, helpStyle.Render(truncateWidth(compositeBottomLine(m, innerW), innerW)))

	content := strings.Join(contentLines, "\n")

	rendered := appBorderStyle.Width(lineWidth).Render(content)

	var overlay strings.Builder
	if m.mode == modeInput {
		overlay.WriteString(renderInputModal(m, lineWidth))
		overlay.WriteString("\n")
	}
	if m.mode == modeConfirm {
		overlay.WriteString(renderConfirmModal(m, lineWidth))
		overlay.WriteString("\n")
	}
	if m.helpOpen {
		overlay.WriteString(renderHelpOverlay(lineWidth))
		overlay.WriteString("\n")
	}

	if overlay.Len() > 0 {
		return rendered + "\n" + overlay.String()
	}
	return rendered
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

func appBorderWidth() int {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2).
		GetHorizontalFrameSize()
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
