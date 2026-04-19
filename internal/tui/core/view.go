package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	metaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	filterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("48"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	listBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Top: "-", Bottom: "-", Left: "|", Right: "|", TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+"}).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)
)

func (m model) View() string {
	m = m.reflow()
	lineWidth := m.width
	if lineWidth <= 0 {
		lineWidth = 80
	}
	lineWidth = max(20, lineWidth)

	var b strings.Builder

	b.WriteString(titleStyle.Render(truncateWidth("hetki tui - live tmux browser", lineWidth)))
	b.WriteString("\n")
	b.WriteString(metaStyle.Render(truncateWidth(contextLine(m.snapshot.ContextBars), lineWidth)))
	b.WriteString("\n")
	b.WriteString(filterStyle.Render(truncateWidth(filterLine(m), lineWidth)))
	b.WriteString("\n")

	start := m.offset
	end := m.offset + m.listH
	if end > len(m.rows) {
		end = len(m.rows)
	}

	innerW := max(16, lineWidth-6)
	lines := make([]string, 0, m.listH)

	if len(m.rows) == 0 {
		lines = append(lines, metaStyle.Render(truncateWidth("no sessions found", innerW)))
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

			active := " "
			if r.Node.Active {
				active = "*"
			}

			line := fmt.Sprintf("%s%s %s", cursor, active, rowLabel(r))
			line = truncateWidth(line, innerW)
			if r.Node.Active {
				line = activeStyle.Render(line)
			}
			if i == m.cursor {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
		for i := end; i < start+m.listH; i++ {
			lines = append(lines, "")
		}
	}

	b.WriteString(listBoxStyle.Width(lineWidth - 2).Render(strings.Join(lines, "\n")))
	b.WriteString("\n")

	statusText := "status: " + m.status + " | mode: " + m.modeLabel()
	if m.mode == modeFilter {
		statusText += " | filter: " + m.filter
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render(truncateWidth(statusText, lineWidth)))
	} else {
		b.WriteString(statusStyle.Render(truncateWidth(statusText, lineWidth)))
	}
	b.WriteString("\n")

	position := "0/0"
	if len(m.rows) > 0 {
		position = fmt.Sprintf("%d/%d", m.cursor+1, len(m.rows))
	}

	help := "keys: j/k move, u/d page, / search, Ctrl+L clear, enter switch, r refresh, q quit"
	b.WriteString(helpStyle.Render(truncateWidth(help+" | row "+position, lineWidth)))
	b.WriteString("\n")

	return b.String()
}

func contextLine(ctx map[string]string) string {
	lines := contextLines(ctx)
	if len(lines) == 0 {
		return "source: live"
	}
	return strings.Join(lines, " | ")
}

func filterLine(m model) string {
	query := strings.TrimSpace(m.filter)
	if m.mode == modeFilter {
		return "filter> " + m.filter + "_"
	}
	if query == "" {
		return "filter: (press / to search)"
	}
	return "filter: " + m.filter
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
