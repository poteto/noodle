package tui

// renderLayout composes the split layout: left rail + tabbed right pane.
func (m Model) renderLayout() string {
	if m.width <= 0 || m.height <= 0 {
		return titleStyle.Render("noodle") + " " + dimStyle.Render("loading...")
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
		rail = renderCompactRail(m.snapshot, m.height)
	} else {
		rail = renderRail(m.snapshot, m.now(), m.height)
	}
	tabBar := renderTabBar(m.activeTab, paneWidth)

	// Tab bar takes ~2 lines; steer/help/status take ~3 at most.
	contentHeight := m.height - 7
	if contentHeight < 4 {
		contentHeight = 4
	}

	var tabContent string
	switch m.activeTab {
	case TabFeed:
		tabContent = m.feedTab.Render(paneWidth, contentHeight, m.now())
	case TabQueue:
		qt := NewQueueTab()
		activeIDs := make([]string, 0, len(m.snapshot.Active))
		for _, s := range m.snapshot.Active {
			activeIDs = append(activeIDs, s.ID)
		}
		qt.SetQueue(m.snapshot.Queue, activeIDs, m.snapshot.ActionNeeded)
		tabContent = qt.Render(paneWidth, contentHeight)
	case TabBrain:
		tabContent = m.brainTab.Render(paneWidth, contentHeight)
	case TabConfig:
		tabContent = dimStyle.Render("(config — coming soon)")
	}

	pane := tabBar + "\n\n" + tabContent
	layout := joinLayout(rail, pane)

	if m.steerOpen {
		layout += "\n\n" + keybarStyle.Render("[steer] ") +
			dimStyle.Render("type @target instruction · enter sends · esc closes")
	}
	if m.helpOpen {
		layout += "\n\n" + renderHelp()
	}
	if m.statusLine != "" {
		layout += "\n" + dimStyle.Render("status: "+m.statusLine)
	}
	if m.err != nil {
		layout += "\n" + errorStyle.Render("error: "+m.err.Error())
	}
	if m.quitPending {
		layout += "\n" + dimStyle.Render("press ctrl+c again to quit")
	}

	return layout
}

func renderHelp() string {
	return sectionStyle.Render("Keys") + "\n" +
		dimStyle.Render("1-4") + " switch tabs  " +
		dimStyle.Render("`") + " steer  " +
		dimStyle.Render("p") + " pause/resume  " +
		dimStyle.Render("m") + " merge  " +
		dimStyle.Render("x") + " reject  " +
		dimStyle.Render("a") + " merge all approved  " +
		dimStyle.Render("?") + " help  " +
		dimStyle.Render("ctrl+c") + " quit"
}
