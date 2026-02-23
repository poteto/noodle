// Package components provides reusable TUI rendering primitives.
// Components are pure render functions — no message handling, no state.
// Every render method takes width int. Parent calculates available width.
package components

import "github.com/charmbracelet/lipgloss"

// Theme holds the pastel color palette shared across all components.
type Theme struct {
	Brand      lipgloss.Color
	Border     lipgloss.Color
	Surface    lipgloss.Color
	CardBG     lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Info       lipgloss.Color
	Text       lipgloss.Color
	Dim        lipgloss.Color
	Secondary  lipgloss.Color
	Execute    lipgloss.Color
	Plan       lipgloss.Color
	Quality    lipgloss.Color
	Reflect    lipgloss.Color
	Prioritize lipgloss.Color
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
