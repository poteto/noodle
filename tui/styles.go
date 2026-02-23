package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds the pastel color palette for the TUI.
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

var theme = Theme{
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

var (
	titleStyle   = lipgloss.NewStyle().Foreground(theme.Brand).Bold(true)
	accentStyle  = lipgloss.NewStyle().Foreground(theme.Brand)
	dimStyle     = lipgloss.NewStyle().Foreground(theme.Dim)
	mutedStyle   = lipgloss.NewStyle().Foreground(theme.Secondary)
	infoStyle    = lipgloss.NewStyle().Foreground(theme.Info)
	costStyle    = lipgloss.NewStyle().Foreground(theme.Brand)
	successStyle = lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(theme.Error).Bold(true)
	sectionStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(theme.Secondary)
	keybarStyle  = lipgloss.NewStyle().Foreground(theme.Brand)

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2a2a40")).
				Foreground(theme.Text)

	railStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Foreground(theme.Text).
			Padding(0, 1)
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
		return mutedStyle.Render(label)
	case "think":
		return dimStyle.Render(label)
	default:
		return dimStyle.Render(label)
	}
}

const noodleBowl = "" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣤⣦⣤⣤⣤⣤⣤⣶⣶⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⡿⠛⢻⠛⢻⠛⢻⣿⡟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣿⡇⠀⡿⠀⣼⠀⢸⣿⡅⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠘⣿⡇⠀⣿⠀⢹⠀⢸⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣠⣤⣤⡀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⡀⠸⡆⠘⣇⠀⢿⣷⠀⠀⠀⠀⣀⣠⣤⣶⣶⣾⣿⠿⠿⠛⠋⢻⡆\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⡇⠀⣿⠀⢿⣄⣸⣿⣦⣤⣴⠿⠿⠛⠛⠉⠁⢀⣀⣀⣀⣄⣤⣼⣿\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣿⡇⠀⡿⠀⣼⣿⣿⣯⣿⣦⣤⣤⣶⣶⣶⣿⢿⠿⠟⠿⠛⠛⠛⠛⠋\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⠁⣸⠃⢠⡟⢻⣿⣿⣿⣿⣿⣭⣭⣭⣵⣶⣤⣀⣄⣠⣤⣤⣴⣶⣦\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⡇⠀⣿⠀⣸⠀⢸⣿⣶⣦⣤⣤⣄⣀⣀⣀⠀⠀⠉⠈⠉⠈⠉⠉⢽⣿\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣸⣿⡇⠀⣿⠀⢸⠀⢸⣿⡿⣿⣿⣿⣿⡟⠛⠻⠿⠿⠿⣿⣶⣶⣶⣶⣿⣿\n" +
	"⠀⠀⠀⠀⠀⠀⠀⢀⣤⣶⣿⡿⣿⣿⣿⣷⠀⠹⡆⠘⣇⠈⣿⡟⠛⠛⠛⠾⣿⡳⣄⠀⠀⠀⠀⠀⠀⠈⠉⠉⠁\n" +
	"⠀⠀⠀⠀⠀⠀⣰⣿⢟⡭⠒⢀⣐⣲⣿⣿⡇⠀⣷⠀⢿⠀⢸⣏⣈⣁⣉⣳⣬⣻⣿⣷⣀⠀⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⣀⣤⣾⣿⡿⠟⠛⠛⠿⣿⣋⣡⠤⢺⡇⠀⡿⠀⣼⠀⢸⣿⠟⠋⣉⢉⡉⣉⠙⠻⢿⣯⣿⣦⣄⠀⠀⠀⠀\n" +
	"⢠⣾⡿⢋⣽⠋⣠⠊⣉⠉⢲⣈⣿⣧⣶⣿⠁⢠⣇⣠⣯⣀⣾⠧⠖⣁⣠⣤⣤⣤⣭⣷⣄⠙⢿⡙⢿⣷⡀⠀⠀\n" +
	"⢸⣿⣄⠸⣧⣼⣁⡎⣠⡾⠛⣉⠀⠄⣈⣉⠻⢿⣋⠁⠌⣉⠻⣧⡾⢋⡡⠔⠒⠒⠢⢌⣻⣶⣾⠇⣸⣿⡇⠀⠀\n" +
	"⣹⣿⣿⣷⣦⣍⣛⠻⠿⠶⢾⣤⣤⣦⣤⣬⣷⣬⣿⣦⣤⣬⣷⣼⣿⣧⣴⣾⠿⠿⠿⢛⣛⣩⣴⣾⣿⣿⡇⠀⠀\n" +
	"⣸⣿⣟⡾⣽⣻⢿⡿⣷⣶⣦⣤⣤⣤⣬⣭⣉⣍⣉⣉⣩⣩⣭⣭⣤⣤⣤⣴⣶⣶⣿⡿⣿⣟⣿⣽⣿⣿⡇⠀⠀\n" +
	"⢸⣿⡍⠉⠛⠛⠿⠽⣷⣯⣿⣽⣻⣻⣟⢿⣻⢿⡿⣿⣟⣿⣻⢟⣿⣻⢯⣿⣽⣾⣷⠿⠗⠛⠉⠁⢸⣿⡇⠀⠀\n" +
	"⠘⣿⣧⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠉⠛⠙⠛⠛⠛⠛⠋⠛⠋⠉⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⣿⡿⠀⠀⠀\n" +
	"⠀⠹⣿⣆⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⣷⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⣦⡀⠀⠀⠀⠀⠀⠀⣼⣿⠇⠀⠀⠀\n" +
	"⠀⠀⠹⣿⣆⠀⠀⠀⠀⠀⠀⠀⠻⠿⠟⠀⠀⠀⠿⣦⣤⠞⠀⠀⠀⠻⠿⠟⠀⠀⠀⠀⠀⢀⣼⣿⠋⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠘⢿⣷⣶⣶⣤⣤⣤⣀⣀⣀⡀⣀⠀⡀⠀⠀⠀⡀⣀⡀⣀⣀⣀⣠⣤⣤⣴⣶⣶⣿⡿⠃⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠙⢿⣿⣾⡙⠯⠿⠽⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠿⠙⢋⣿⣿⡿⠋⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠙⠻⢿⣶⣤⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣤⣾⣿⠿⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀\n" +
	"⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣈⠙⠻⠿⡿⠷⠶⡶⠶⠶⠶⠶⠶⣾⢿⣿⢿⠛⣉⣡⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"
