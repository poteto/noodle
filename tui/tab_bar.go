package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Tab identifies which tab is active in the right pane.
type Tab int

const (
	TabFeed Tab = iota
	TabQueue
	TabConfig
)

var tabNames = [3]string{"Feed", "Queue", "Config"}

func (t Tab) String() string {
	if int(t) >= 0 && int(t) < len(tabNames) {
		return tabNames[t]
	}
	return "?"
}

// renderTabBar renders the horizontal tab bar with the active tab highlighted.
func renderTabBar(active Tab, width int) string {
	gap := "   "
	var top, bot strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			top.WriteString(gap)
			bot.WriteString(strings.Repeat(" ", len(gap)))
		}
		w := lipgloss.Width(name)
		if Tab(i) == active {
			top.WriteString(accentStyle.Bold(true).Render(name))
			bot.WriteString(accentStyle.Render(strings.Repeat("─", w)))
		} else {
			top.WriteString(dimStyle.Render(name))
			bot.WriteString(strings.Repeat(" ", w))
		}
	}
	return top.String() + "\n" + bot.String()
}
