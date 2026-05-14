package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type SearchBarProps struct {
	Width  int
	Filter string
	Active bool
	Style  lipgloss.Style
}

func RenderSearchBar(props SearchBarProps) string {
	search := strings.TrimSpace(props.Filter)
	prompt := "\uf002 "
	if props.Active {
		return props.Style.Render(truncateWidth(prompt+search+"_", props.Width))
	}
	if search != "" {
		return props.Style.Render(truncateWidth(prompt+search, props.Width))
	}
	return props.Style.Render(prompt)
}
