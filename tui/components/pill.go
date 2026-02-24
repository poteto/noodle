package components

import "image/color"
import "charm.land/lipgloss/v2"

// Pill renders an inline bordered button like ┌─ Label ─┐.
type Pill struct {
	Label   string
	Icon    string
	Color   color.Color
	Focused bool
}

// Render returns the pill constrained to available width.
func (p Pill) Render(width int) string {
	t := DefaultTheme
	color := p.Color
	if color == nil {
		color = t.Secondary
	}

	label := p.Label
	if p.Icon != "" {
		label = p.Icon + " " + label
	}

	borderColor := t.Dim
	fg := t.Secondary
	if p.Focused {
		borderColor = color
		fg = t.Text
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Foreground(fg).
		Padding(0, 1)

	return style.Render(label)
}
