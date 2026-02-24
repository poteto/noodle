package components

import "image/color"
import "charm.land/lipgloss/v2"

// Badge renders a colored inline label like APPROVE or FLAG.
func Badge(label string, color color.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(label)
}

// TaskTypeBadge returns a badge colored by task type.
func TaskTypeBadge(taskType string) string {
	return Badge(taskType, TaskTypeColor(taskType))
}

// VerdictBadge returns a badge colored by verdict outcome.
func VerdictBadge(verdict string) string {
	t := DefaultTheme
	switch verdict {
	case "APPROVE":
		return Badge(verdict, t.Success)
	case "FLAG":
		return Badge(verdict, t.Warning)
	case "REJECT":
		return Badge(verdict, t.Error)
	default:
		return Badge(verdict, t.Dim)
	}
}
