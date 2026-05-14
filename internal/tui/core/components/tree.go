package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MSmaili/hetki/internal/tui/contracts"
)

type TreeStyles struct {
	Meta        lipgloss.Style
	Row         lipgloss.Style
	SessionRow  lipgloss.Style
	WindowRow   lipgloss.Style
	ActiveRow   lipgloss.Style
	SelectedRow lipgloss.Style
	Rail        lipgloss.Style
}

type TreeRowProps struct {
	NodeID      string
	Kind        contracts.NodeKind
	Label       string
	Depth       int
	TreePrefix  string
	Expanded    bool
	Branch      bool
	Active      bool
	Selected    bool
	WindowCount int
}

type TreeProps struct {
	Width     int
	EmptyText string
	Rows      []TreeRowProps
	Styles    TreeStyles
}

func RenderTree(props TreeProps) []string {
	if len(props.Rows) == 0 {
		return []string{props.Styles.Meta.Render(truncateWidth(props.EmptyText, props.Width))}
	}

	lines := make([]string, 0, len(props.Rows))
	for _, row := range props.Rows {
		line := renderRowLine(row, props.Styles)
		line = truncateWidth(line, props.Width)
		lines = append(lines, styleRowLine(row, line, props.Width, props.Styles))
	}
	return lines
}

func renderRowLine(row TreeRowProps, styles TreeStyles) string {
	cursor := " "
	if row.Selected {
		cursor = "❯"
	}

	marker := "  "
	if row.Active {
		marker = "● "
	}

	label := decoratedLabel(row, styles)
	if row.WindowCount > 0 {
		label += " " + styles.Meta.Render(fmt.Sprintf("(%d)", row.WindowCount))
	}
	return fmt.Sprintf("%s %s%s", cursor, marker, label)
}

func styleRowLine(row TreeRowProps, line string, width int, styles TreeStyles) string {
	if row.Selected {
		return styles.SelectedRow.Width(width).Render(line)
	}
	if row.Kind == contracts.NodeKindSession {
		return styles.SessionRow.Render(line)
	}
	if row.Active {
		return styles.ActiveRow.Render(line)
	}
	if row.Kind == contracts.NodeKindWindow {
		return styles.WindowRow.Render(line)
	}
	return styles.Row.Render(line)
}

func decoratedLabel(row TreeRowProps, styles TreeStyles) string {
	label := strings.TrimSpace(row.Label)
	if label == "" {
		label = row.NodeID
	}
	prefix := row.TreePrefix
	branch := branchGlyph(row)
	if !row.Selected {
		prefix = styles.Rail.Render(prefix)
		branch = styles.Rail.Render(branch)
	}
	if row.Depth == 0 {
		return fmt.Sprintf("%s %s %s", branch, nodeIcon(row.Kind), label)
	}
	return fmt.Sprintf("  %s%s %s", prefix, nodeIcon(row.Kind), label)
}

func branchGlyph(row TreeRowProps) string {
	if !row.Branch {
		return "  "
	}
	if row.Expanded {
		return "\uf0d7 "
	}
	return "\uf0da "
}

func nodeIcon(kind contracts.NodeKind) string {
	if kind == contracts.NodeKindSession {
		return "\U000f018d"
	}
	return "\ueb14"
}
