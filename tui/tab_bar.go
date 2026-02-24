package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Tab identifies which tab is active in the right pane.
type Tab int

const (
	TabFeed Tab = iota
	TabQueue
	TabReviews
)

func (t Tab) String() string {
	switch t {
	case TabFeed:
		return "Feed"
	case TabQueue:
		return "Queue"
	case TabReviews:
		return "Reviews"
	}
	return "?"
}

// renderTabBar renders the horizontal tab bar with the active tab highlighted.
func renderTabBar(active Tab, width int, pendingReviewCount int) string {
	gap := "   "
	var top, bot strings.Builder
	tabs := []Tab{TabFeed, TabQueue, TabReviews}
	for i, tab := range tabs {
		name := tab.String()
		if tab == TabReviews && pendingReviewCount > 0 {
			name = "Reviews (" + fmt.Sprintf("%d", pendingReviewCount) + ")"
		}
		if i > 0 {
			top.WriteString(gap)
			bot.WriteString(strings.Repeat(" ", len(gap)))
		}
		w := lipgloss.Width(name)
		if tab == active {
			top.WriteString(accentStyle.Bold(true).Render(name))
			bot.WriteString(accentStyle.Render(strings.Repeat("─", w)))
		} else {
			top.WriteString(dimStyle.Render(name))
			bot.WriteString(strings.Repeat(" ", w))
		}
	}
	return top.String() + "\n" + bot.String()
}
