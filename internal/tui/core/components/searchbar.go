package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type SearchBarProps struct {
	Width       int
	Filter      string
	Active      bool
	Compact     bool
	Style       lipgloss.Style
	PromptStyle lipgloss.Style
}

func RenderSearchBar(props SearchBarProps) string {
	search := strings.TrimSpace(props.Filter)
	prompt := "\uf002 "
	if props.Compact && search == "" && !props.Active {
		prompt = "\uf002 search"
	}
	content := prompt + search
	if props.Active {
		content += "_"
	} else if search == "" {
		content = prompt
	}
	if props.Width <= 0 {
		return ""
	}
	content = truncateWidth(content, props.Width)
	if strings.HasPrefix(content, prompt) {
		rest := strings.TrimPrefix(content, prompt)
		return props.PromptStyle.Render(prompt) + props.Style.Render(rest)
	}
	return props.Style.Render(content)
}
