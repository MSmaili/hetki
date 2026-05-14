package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type InputModalProps struct {
	LineWidth  int
	Title      string
	Prompt     string
	Value      string
	ModalStyle lipgloss.Style
	TitleStyle lipgloss.Style
	HintStyle  lipgloss.Style
}

func RenderInputModal(props InputModalProps) string {
	content := strings.Join([]string{
		props.TitleStyle.Render(props.Title),
		props.Prompt,
		"> " + props.Value + "_",
		props.HintStyle.Render("enter submit | esc cancel"),
	}, "\n")
	return props.ModalStyle.Width(min(72, props.LineWidth-8)).Render(content)
}
