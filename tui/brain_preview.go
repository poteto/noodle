package tui

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders markdown content as styled terminal output.
func RenderMarkdown(content string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}
