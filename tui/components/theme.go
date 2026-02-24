// Package components provides reusable TUI rendering primitives.
// Components are pure render functions — no message handling, no state.
// Every render method takes width int. Parent calculates available width.
package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme holds the pastel color palette shared across all components.
type Theme struct {
	Brand      color.Color
	Border     color.Color
	Surface    color.Color
	CardBG     color.Color
	Success    color.Color
	Warning    color.Color
	Error      color.Color
	Info       color.Color
	Text       color.Color
	Dim        color.Color
	Secondary  color.Color
	Execute    color.Color
	Plan       color.Color
	Quality    color.Color
	Reflect    color.Color
	Prioritize color.Color
}

// DefaultTheme is the pastel palette used by Noodle's TUI.
var DefaultTheme = Theme{
	Brand:      lipgloss.Color("#fde68a"),
	Border:     lipgloss.Color("#fcd34d"),
	Surface:    lipgloss.Color("#1c1c2e"),
	CardBG:     lipgloss.Color("#24243a"),
	Success:    lipgloss.Color("#86efac"),
	Warning:    lipgloss.Color("#fdba74"),
	Error:      lipgloss.Color("#fca5a5"),
	Info:       lipgloss.Color("#93c5fd"),
	Text:       lipgloss.Color("#f5f5f5"),
	Dim:        lipgloss.Color("#6b6b80"),
	Secondary:  lipgloss.Color("#a8a8b8"),
	Execute:    lipgloss.Color("#86efac"),
	Plan:       lipgloss.Color("#93c5fd"),
	Quality:    lipgloss.Color("#fde68a"),
	Reflect:    lipgloss.Color("#f9a8d4"),
	Prioritize: lipgloss.Color("#fdba74"),
}

// ColorPool is a set of distinct pastel colors for hashing arbitrary task types.
// The first 5 match the named task type colors for consistency.
var ColorPool = []color.Color{
	lipgloss.Color("#86efac"), // green (execute)
	lipgloss.Color("#93c5fd"), // blue (plan)
	lipgloss.Color("#fde68a"), // yellow (quality)
	lipgloss.Color("#f9a8d4"), // pink (reflect)
	lipgloss.Color("#fdba74"), // orange (prioritize)
	lipgloss.Color("#c4b5fd"), // violet
	lipgloss.Color("#67e8f9"), // cyan
	lipgloss.Color("#fda4af"), // rose
	lipgloss.Color("#a3e635"), // lime
	lipgloss.Color("#fbbf24"), // amber
	lipgloss.Color("#818cf8"), // indigo
	lipgloss.Color("#34d399"), // emerald
	lipgloss.Color("#fb923c"), // tangerine
	lipgloss.Color("#e879f9"), // fuchsia
	lipgloss.Color("#38bdf8"), // sky
	lipgloss.Color("#a78bfa"), // purple
}

// TaskTypeColor returns a consistent color for a task type. Named types use
// their dedicated theme color; unknown types hash into the color pool.
func TaskTypeColor(taskType string) color.Color {
	switch strings.ToLower(taskType) {
	case "execute":
		return DefaultTheme.Execute
	case "plan":
		return DefaultTheme.Plan
	case "quality":
		return DefaultTheme.Quality
	case "reflect":
		return DefaultTheme.Reflect
	case "prioritize":
		return DefaultTheme.Prioritize
	}
	h := uint32(0)
	for _, c := range taskType {
		h = h*31 + uint32(c)
	}
	return ColorPool[h%uint32(len(ColorPool))]
}
