package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type StatusBarProps struct {
	Width       int
	Status      string
	Center      string
	Right       string
	Compact     bool
	StatusStyle lipgloss.Style
	HelpStyle   lipgloss.Style
	MetaStyle   lipgloss.Style
}

func RenderStatusBar(props StatusBarProps) string {
	if props.Width <= 0 {
		return ""
	}
	leftStyled := props.StatusStyle.Render(props.Status)
	rightStyled := props.HelpStyle.Render(props.Right)
	centerStyled := props.MetaStyle.Render(props.Center)

	leftW := lipgloss.Width(leftStyled)
	rightW := lipgloss.Width(rightStyled)
	centerW := lipgloss.Width(centerStyled)

	if props.Compact || leftW+centerW+rightW+4 > props.Width {
		centerStyled = ""
		centerW = 0
	}
	if leftW+rightW+2 > props.Width {
		leftStyled = props.StatusStyle.Render(truncateWidth(props.Status, props.Width-rightW-2))
		leftW = lipgloss.Width(leftStyled)
	}

	if centerW == 0 {
		gap := props.Width - leftW - rightW
		if gap < 1 {
			gap = 1
		}
		return leftStyled + strings.Repeat(" ", gap) + rightStyled
	}

	gap := props.Width - leftW - centerW - rightW
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

func RenderPositionLabel(cursor, total int, base string) string {
	if total <= 0 {
		return base
	}
	return fmt.Sprintf("%d/%d · %s", cursor, total, base)
}

func RenderMatchLabel(current, total int, base string) string {
	if total <= 0 {
		return base
	}
	return fmt.Sprintf("match %d/%d · %s", current, total, base)
}
