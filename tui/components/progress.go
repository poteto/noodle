package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a budget progress bar with pastel yellow fill
// that shifts to orange above 75%.
func ProgressBar(current, total float64, width int) string {
	t := DefaultTheme
	if width < 10 {
		width = 10
	}

	pct := 0.0
	if total > 0 {
		pct = current / total
		if pct > 1 {
			pct = 1
		}
	}

	barWidth := width - 8 // room for percentage label
	if barWidth < 4 {
		barWidth = 4
	}

	filled := int(pct * float64(barWidth))
	empty := barWidth - filled

	fillColor := t.Brand
	if pct > 0.75 {
		fillColor = t.Warning
	}
	if pct > 0.95 {
		fillColor = t.Error
	}

	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(t.Dim)
	label := fmt.Sprintf(" %3.0f%%", pct*100)

	return fillStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty)) +
		lipgloss.NewStyle().Foreground(t.Secondary).Render(label)
}
