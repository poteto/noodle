package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme holds the pastel color palette for the TUI.
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
	case "idle":
		return dimStyle.Render("idle")
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
	case "dropped":
		return warnStyle.Render(label)
	case "rebuild":
		return dimStyle.Render(label)
	case "bootstrap":
		return accentStyle.Render(label)
	default:
		return dimStyle.Render(label)
	}
}

// wrapText wraps s to maxWidth using word boundaries. Long words that exceed
// maxWidth are broken at the width boundary. Returns a slice of lines.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		if strings.TrimSpace(paragraph) == "" {
			lines = append(lines, "")
			continue
		}
		// Preserve leading whitespace so indented content (JSON, code)
		// keeps its structure.
		trimmed := strings.TrimLeft(paragraph, " \t")
		indent := paragraph[:len(paragraph)-len(trimmed)]
		indentWidth := len(indent)

		words := strings.Fields(trimmed)
		wrapWidth := maxWidth - indentWidth
		if wrapWidth < 10 {
			wrapWidth = 10
		}
		var line string
		for _, word := range words {
			// Break long words.
			for len(word) > wrapWidth {
				if line != "" {
					lines = append(lines, indent+line)
					line = ""
				}
				lines = append(lines, indent+word[:wrapWidth])
				word = word[wrapWidth:]
			}
			if line == "" {
				line = word
			} else if len(line)+1+len(word) <= wrapWidth {
				line += " " + word
			} else {
				lines = append(lines, indent+line)
				line = word
			}
		}
		if line != "" {
			lines = append(lines, indent+line)
		}
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// renderEmptyState renders the noodle bowl art with a message, centered
// horizontally and vertically within the given dimensions.
// Falls back to text-only when the viewport is too short for the bowl art.
func renderEmptyState(msg string, width, height int) string {
	bowl := dimStyle.Render(noodleBowl)
	text := dimStyle.Render(msg)
	content := bowl + "\n\n" + text
	if strings.Count(content, "\n")+1 > height {
		content = text
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
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
