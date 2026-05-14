package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type ConfirmModalProps struct {
	LineWidth  int
	Title      string
	Body       string
	ModalStyle lipgloss.Style
	TitleStyle lipgloss.Style
	HintStyle  lipgloss.Style
}

func RenderConfirmModal(props ConfirmModalProps) string {
	content := strings.Join([]string{
		props.TitleStyle.Render(props.Title),
		props.Body,
		props.HintStyle.Render("enter/y confirm | esc/n cancel"),
	}, "\n")
	return props.ModalStyle.Width(min(72, props.LineWidth-8)).Render(content)
}
