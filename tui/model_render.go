package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderLayout composes the split layout: left rail + tabbed right pane + keybar.
func (m Model) renderLayout() string {
	if m.width <= 0 || m.height <= 0 {
		return titleStyle.Render("noodle") + " " + dimStyle.Render("loading...")
	}

	// Reserve bottom lines for keybar (1) + status/error (1).
	bottomReserve := 2
	layoutHeight := m.height - bottomReserve
	if layoutHeight < 6 {
		layoutHeight = 6
	}

	// Feed tab uses the full width (no rail). Other tabs show the rail.
	showRail := m.activeTab != TabFeed
	var paneWidth int
	if showRail {
		compact := m.width < 80
		effectiveRailWidth := railWidth
		if compact {
			effectiveRailWidth = 8
		}
		paneWidth = m.width - effectiveRailWidth - 1
	} else {
		paneWidth = m.width
	}
	if paneWidth < 20 {
		paneWidth = 20
	}

	tabBar := renderTabBar(m.activeTab, paneWidth, m.snapshot.PendingReviewCount)

	// Feed: 100% - tab bar - keybar. Other tabs: additional padding.
	contentHeight := layoutHeight - 2 // tab bar (1 line) + 1 line gap
	if m.activeTab != TabFeed {
		contentHeight = layoutHeight - 4
	}
	if contentHeight < 4 {
		contentHeight = 4
	}

	var tabContent string
	if m.taskEditor.open {
		editorWidth := paneWidth - 4
		if editorWidth < 30 {
			editorWidth = 30
		}
		tabContent = lipgloss.Place(paneWidth, contentHeight,
			lipgloss.Center, lipgloss.Center,
			m.taskEditor.Render(editorWidth))
	} else if m.detailSession != "" {
		tabContent = m.renderActorDetail(paneWidth, contentHeight)
	} else {
		switch m.activeTab {
		case TabFeed:
			tabContent = m.feedTab.Render(paneWidth, contentHeight, m.now())
		case TabQueue:
			tabContent = m.queueTab.Render(paneWidth, contentHeight)
		case TabReviews:
			tabContent = m.reviewsTab.Render(paneWidth, contentHeight)
		}
	}

	pane := tabBar + "\n" + tabContent
	var layout string
	if showRail {
		compact := m.width < 80
		var rail string
		if compact {
			rail = renderCompactRail(m.snapshot, layoutHeight)
		} else {
			rail = renderRail(m.snapshot, layoutHeight, m.shimmerIndex)
		}
		layout = joinLayout(rail, pane)
	} else {
		layout = pane
	}

	if m.steerOpen {
		inputLine := keybarStyle.Render("[steer] ") + m.steerInput + keybarStyle.Render("█")
		if m.steerMentionOpen && len(m.steerMentionItems) > 0 {
			for i, item := range m.steerMentionItems {
				if i == m.steerMentionIndex {
					inputLine += "\n  " + keybarStyle.Render("> "+item)
				} else {
					inputLine += "\n  " + dimStyle.Render("  "+item)
				}
			}
		}
		if m.steerInput == "" {
			inputLine += " " + dimStyle.Render("@target instruction · enter sends · esc closes")
		}
		layout += "\n" + inputLine
	}

	// Persistent keybar.
	layout += "\n" + renderKeybar(m.activeTab, m.detailSession != "", m.detailAutoScroll)

	// Status / error line.
	var bottom string
	if m.err != nil {
		bottom = errorStyle.Render("error: " + m.err.Error())
	} else if m.statusLine != "" {
		bottom = dimStyle.Render("status: " + m.statusLine)
	} else if m.quitPending {
		bottom = warnStyle.Render("ctrl+c again to quit — this will kill all running agents")
	}
	layout += "\n" + bottom

	return layout
}

// renderKeybar returns a single-line context-sensitive shortcut bar.
func renderKeybar(tab Tab, inDetail bool, autoScroll bool) string {
	if inDetail {
		parts := []string{
			dimStyle.Render("esc") + " back",
			dimStyle.Render("h/l") + " prev/next",
			dimStyle.Render("j/k") + " scroll",
		}
		if autoScroll {
			parts = append(parts, successStyle.Render("auto-scroll"))
		} else {
			parts = append(parts, warnStyle.Render("manual"))
		}
		return strings.Join(parts, "  ")
	}

	var parts []string
	parts = append(parts, dimStyle.Render("1-3")+" tabs")

	switch tab {
	case TabFeed:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" open",
		)
	case TabQueue:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" open",
		)
	case TabReviews:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" view diff",
			dimStyle.Render("m")+" merge",
			dimStyle.Render("x")+" reject",
			dimStyle.Render("c")+" request changes",
		)
	}

	if tab == TabFeed || tab == TabQueue {
		parts = append(parts, dimStyle.Render("h/l")+" actors")
	}

	parts = append(parts,
		dimStyle.Render("`")+" steer",
		dimStyle.Render("n")+" new task",
		dimStyle.Render("p")+" pause",
	)

	return strings.Join(parts, "  ")
}

// renderActorDetail renders the session detail (actor) view with a scrollable
// column layout. Events are chronological (oldest first) with word-wrapped
// bodies. Supports auto-scroll to newest and manual scroll via j/k.
func (m Model) renderActorDetail(width, height int) string {
	sid := m.detailSession

	var session *Session
	for i := range m.snapshot.Sessions {
		if m.snapshot.Sessions[i].ID == sid {
			session = &m.snapshot.Sessions[i]
			break
		}
	}
	if session == nil {
		return dimStyle.Render(fmt.Sprintf("session %q not found", sid))
	}

	// Header: name + metadata (fixed, 3 lines).
	name := session.DisplayName
	if name == "" {
		name = session.ID
	}
	header := titleStyle.Render(name) + "\n"

	meta := []string{
		statusLabel(session.Status),
		dimStyle.Render(shortModelName(session.Model)),
		dimStyle.Render(durationLabel(session.DurationSeconds)),
	}
	if session.TotalCostUSD > 0 {
		meta = append(meta, costStyle.Render(fmt.Sprintf("$%.4f", session.TotalCostUSD)))
	}
	if session.ContextWindowUsagePct > 0 {
		meta = append(meta, dimStyle.Render(fmt.Sprintf("ctx %.0f%%", session.ContextWindowUsagePct*100)))
	}
	header += strings.Join(meta, "  ") + "\n"

	// Event area height.
	eventHeight := height - 3
	if eventHeight < 3 {
		eventHeight = 3
	}

	events := m.snapshot.EventsBySession[sid]
	if len(events) == 0 {
		return header + "\n" + dimStyle.Render("(no events)")
	}

	// Column layout: HH:MM:SS  LABEL     body text
	//                                     continuation
	const tsWidth = 8
	const labelWidth = 10
	const gap = 2
	prefixWidth := tsWidth + gap + labelWidth + gap
	msgWidth := width - prefixWidth
	if msgWidth < 20 {
		msgWidth = 20
	}
	indent := strings.Repeat(" ", prefixWidth)

	// Render all events into lines.
	var allLines []string
	for _, ev := range events {
		ts := dimStyle.Render(ev.At.Format("15:04:05"))
		label := fmt.Sprintf("%-*s", labelWidth, ev.Label)
		label = eventLabel(label)

		bodyLines := renderEventBody(ev.Body, msgWidth)
		first := ts + strings.Repeat(" ", gap) + label + strings.Repeat(" ", gap) + bodyLines[0]
		allLines = append(allLines, first)
		for _, cont := range bodyLines[1:] {
			allLines = append(allLines, indent+cont)
		}
	}

	totalLines := len(allLines)

	// Compute effective scroll offset.
	scroll := m.detailScroll
	if m.detailAutoScroll {
		scroll = totalLines - eventHeight
	}
	maxScroll := totalLines - eventHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Slice visible window.
	end := scroll + eventHeight
	if end > totalLines {
		end = totalLines
	}
	visible := allLines[scroll:end]

	return header + "\n" + strings.Join(visible, "\n")
}
