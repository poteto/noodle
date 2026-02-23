package tui

// renderSurface returns a placeholder view while the TUI is being rebuilt.
// Phase 2 replaces this with the split layout (rail + tabbed pane).
func (m Model) renderSurface() string {
	return titleStyle.Render("noodle") + " " + dimStyle.Render("command center coming soon") + "\n\n" +
		keybarStyle.Render("s steer | p pause/resume | ? help | ctrl+c quit")
}
