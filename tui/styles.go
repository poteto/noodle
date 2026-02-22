package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent       = lipgloss.Color("#fde68a")
	colorSuccess      = lipgloss.Color("#34d399")
	colorWarn         = lipgloss.Color("#fbbf24")
	colorError        = lipgloss.Color("#f87171")
	colorInfo         = lipgloss.Color("#7dd3fc")
	colorFg           = lipgloss.Color("#d4d4d4")
	colorMuted        = lipgloss.Color("#a3a3a3")
	colorDim          = lipgloss.Color("#737373")
	colorBorder       = lipgloss.Color("#665c00")
	colorSelected     = lipgloss.Color("#525252")
	colorNoodleYellow = lipgloss.Color("#F4E38E")
)

var (
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Foreground(colorFg).
			Padding(0, 1)
	titleStyle   = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	accentStyle  = lipgloss.NewStyle().Foreground(colorAccent)
	dimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	infoStyle    = lipgloss.NewStyle().Foreground(colorInfo)
	costStyle    = lipgloss.NewStyle().Foreground(colorNoodleYellow)
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)
	selectedRowStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("#f5f5f5"))
	sectionStyle = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	keybarStyle  = lipgloss.NewStyle().Foreground(colorAccent)
)

func sectionLine(label string, width int) string {
	if width < 18 {
		width = 18
	}
	title := sectionStyle.Render(label)
	if width <= lipgloss.Width(label)+4 {
		return title
	}
	pad := width - lipgloss.Width(label) - 2
	if pad < 0 {
		pad = 0
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, "— ", title, " ", dimStyle.Render(strings.Repeat("─", pad)))
}

func loopStateLabel(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "paused":
		return warnStyle.Render("paused")
	case "draining":
		return errorStyle.Render("draining")
	default:
		return successStyle.Render("running")
	}
}

func statusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "stuck":
		return errorStyle.Render(status)
	case "completed", "exited":
		return successStyle.Render(status)
	case "paused", "draining":
		return warnStyle.Render(status)
	default:
		return mutedStyle.Render(status)
	}
}

func eventLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "cost":
		return costStyle.Render(label)
	case "ticket":
		return accentStyle.Render(label)
	case "read", "edit", "bash", "glob", "grep":
		return infoStyle.Render(label)
	case "think":
		return mutedStyle.Render(label)
	default:
		return dimStyle.Render(label)
	}
}
