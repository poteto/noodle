package components

import "github.com/charmbracelet/lipgloss"

// Badge renders a colored inline label like APPROVE or FLAG.
func Badge(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(label)
}

// TaskTypeBadge returns a badge colored by task type.
func TaskTypeBadge(taskType string) string {
	t := DefaultTheme
	color := t.Text
	switch taskType {
	case "execute", "Execute":
		color = t.Execute
	case "plan", "Plan":
		color = t.Plan
	case "quality", "Quality":
		color = t.Quality
	case "reflect", "Reflect":
		color = t.Reflect
	case "prioritize", "Prioritize":
		color = t.Prioritize
	}
	return Badge(taskType, color)
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
