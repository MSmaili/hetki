package components

import "charm.land/lipgloss/v2"

func truncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	visualW := lipgloss.Width(s)
	if visualW <= width {
		return s
	}
	r := []rune(s)
	if width <= 3 {
		return string(r[:width])
	}
	lipglossWidth := 0
	i := 0
	for i < len(r) {
		segW := lipgloss.Width(string(r[i]))
		if lipglossWidth+segW > width-3 {
			break
		}
		lipglossWidth += segW
		i++
	}
	return string(r[:i]) + "..."
}
