package tui

import (
	"fmt"
	"strings"
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

	compact := m.width < 80
	effectiveRailWidth := railWidth
	if compact {
		effectiveRailWidth = 8
	}

	paneWidth := m.width - effectiveRailWidth - 1
	if paneWidth < 20 {
		paneWidth = 20
	}

	var rail string
	if compact {
		rail = renderCompactRail(m.snapshot, layoutHeight)
	} else {
		rail = renderRail(m.snapshot, m.now(), layoutHeight, m.shimmerIndex)
	}
	tabBar := renderTabBar(m.activeTab, paneWidth)

	// Tab bar takes 2 lines; leave room for padding.
	contentHeight := layoutHeight - 4
	if contentHeight < 4 {
		contentHeight = 4
	}

	var tabContent string
	if m.detailSession != "" {
		tabContent = m.renderActorDetail(paneWidth, contentHeight)
	} else {
		switch m.activeTab {
		case TabFeed:
			tabContent = m.feedTab.Render(paneWidth, contentHeight, m.now())
		case TabQueue:
			tabContent = m.queueTab.Render(paneWidth, contentHeight)
		case TabBrain:
			tabContent = m.brainTab.Render(paneWidth, contentHeight)
		case TabConfig:
			tabContent = m.configTab.Render(paneWidth, contentHeight)
		}
	}

	pane := tabBar + "\n\n" + tabContent
	layout := joinLayout(rail, pane)

	if m.taskEditor.open {
		layout += "\n" + m.taskEditor.Render(paneWidth)
	}
	if m.steerOpen {
		layout += "\n" + keybarStyle.Render("[steer] ") +
			dimStyle.Render("type @target instruction · enter sends · esc closes")
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
		bottom = dimStyle.Render("press ctrl+c again to quit")
	}
	layout += "\n" + bottom

	return layout
}

// renderKeybar returns a single-line context-sensitive shortcut bar.
func renderKeybar(tab Tab, inDetail bool, autoScroll bool) string {
	if inDetail {
		parts := []string{
			dimStyle.Render("esc") + " back",
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
	parts = append(parts, dimStyle.Render("1-4")+" tabs")

	switch tab {
	case TabFeed:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" open",
			dimStyle.Render("m")+" merge",
			dimStyle.Render("x")+" reject",
			dimStyle.Render("a")+" merge all",
		)
	case TabQueue:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" open",
		)
	case TabBrain:
		parts = append(parts,
			dimStyle.Render("j/k")+" select",
			dimStyle.Render("enter")+" preview",
			dimStyle.Render("esc")+" back",
		)
	case TabConfig:
		parts = append(parts, dimStyle.Render("←/→")+" autonomy")
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

		wrapped := wrapText(ev.Body, msgWidth)
		first := ts + strings.Repeat(" ", gap) + label + strings.Repeat(" ", gap) + dimStyle.Render(wrapped[0])
		allLines = append(allLines, first)
		for _, cont := range wrapped[1:] {
			allLines = append(allLines, indent+dimStyle.Render(cont))
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
