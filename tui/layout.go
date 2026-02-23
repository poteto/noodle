package tui

import "github.com/charmbracelet/lipgloss"

// joinLayout places the rail on the left and the pane on the right with a gap.
func joinLayout(rail, pane string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, rail, " ", pane)
}
