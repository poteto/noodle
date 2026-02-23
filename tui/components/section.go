package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Section renders a section header: Title ╌╌╌╌╌╌╌╌
func Section(title string, width int) string {
	t := DefaultTheme
	titleStyle := lipgloss.NewStyle().Foreground(t.Secondary).Bold(true)
	lineStyle := lipgloss.NewStyle().Foreground(t.Dim)

	rendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(rendered)
	remaining := width - titleWidth - 1
	if remaining <= 0 {
		return rendered
	}
	return rendered + " " + lineStyle.Render(strings.Repeat("╌", remaining))
}
