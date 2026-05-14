package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type HelpEntry struct {
	Keys string
	Desc string
}

type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

type HelpOverlayProps struct {
	LineWidth    int
	Title        string
	Hint         string
	Sections     []HelpSection
	OverlayStyle lipgloss.Style
	TitleStyle   lipgloss.Style
	MetaStyle    lipgloss.Style
	KeyStyle     lipgloss.Style
	HintStyle    lipgloss.Style
}

func RenderHelpOverlay(props HelpOverlayProps) string {
	keyColW := 0
	for _, s := range props.Sections {
		for _, e := range s.Entries {
			if w := lipgloss.Width(e.Keys); w > keyColW {
				keyColW = w
			}
		}
	}

	var lines []string
	lines = append(lines, props.TitleStyle.Render(props.Title))
	for _, s := range props.Sections {
		lines = append(lines, "")
		lines = append(lines, props.MetaStyle.Render(s.Title))
		for _, e := range s.Entries {
			pad := keyColW - lipgloss.Width(e.Keys)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, "  "+props.KeyStyle.Render(e.Keys)+strings.Repeat(" ", pad)+"  "+e.Desc)
		}
	}
	lines = append(lines, "")
	lines = append(lines, props.HintStyle.Render(props.Hint))

	return props.OverlayStyle.Width(min(76, props.LineWidth-8)).Render(strings.Join(lines, "\n"))
}
