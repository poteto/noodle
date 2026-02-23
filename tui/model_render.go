package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	layout += "\n" + renderKeybar(m.activeTab, m.detailSession != "")

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
func renderKeybar(tab Tab, inDetail bool) string {
	if inDetail {
		return dimStyle.Render("esc") + " back"
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

// renderActorDetail renders the session detail (actor) view.
func (m Model) renderActorDetail(width, height int) string {
	sid := m.detailSession

	// Find the session in snapshot.
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

	var b strings.Builder

	name := session.DisplayName
	if name == "" {
		name = session.ID
	}
	b.WriteString(titleStyle.Render(name))
	b.WriteString("\n")

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
	b.WriteString(strings.Join(meta, "  "))
	b.WriteString("\n\n")

	b.WriteString(sectionStyle.Render("Events"))
	b.WriteString("\n")

	events := m.snapshot.EventsBySession[sid]
	if len(events) == 0 {
		b.WriteString(dimStyle.Render("(no events)"))
	} else {
		// Show events newest-first, limited to fit.
		maxEvents := height - 5
		if maxEvents < 3 {
			maxEvents = 3
		}
		start := 0
		if len(events) > maxEvents {
			start = len(events) - maxEvents
		}
		innerWidth := width - 2
		for i := len(events) - 1; i >= start; i-- {
			ev := events[i]
			label := eventLabel(ev.Label)
			body := ev.Body
			if lipgloss.Width(body) > innerWidth-lipgloss.Width(ev.Label)-2 {
				body = body[:innerWidth-lipgloss.Width(ev.Label)-4] + "…"
			}
			b.WriteString(label + " " + dimStyle.Render(body))
			b.WriteString("\n")
		}
	}

	return b.String()
}
