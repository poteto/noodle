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
		rail = renderRail(m.snapshot, m.now(), m.height, m.shimmerIndex)
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
		tabContent = m.queueTab.Render(paneWidth, contentHeight)
	case TabBrain:
		tabContent = m.brainTab.Render(paneWidth, contentHeight)
	case TabConfig:
		tabContent = dimStyle.Render("(config — coming soon)")
	}

	pane := tabBar + "\n\n" + tabContent
	layout := joinLayout(rail, pane)

	if m.taskEditor.open {
		layout += "\n\n" + m.taskEditor.Render(paneWidth)
	}
	if m.steerOpen {
		layout += "\n\n" + keybarStyle.Render("[steer] ") +
			dimStyle.Render("type @target instruction · enter sends · esc closes")
	}
	if m.helpOpen {
		layout += "\n\n" + renderHelp(m.activeTab)
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

func renderHelp(tab Tab) string {
	global := dimStyle.Render("1-4") + " switch tabs  " +
		dimStyle.Render("`") + " steer  " +
		dimStyle.Render("n") + " new task  " +
		dimStyle.Render("p") + " pause/resume  " +
		dimStyle.Render("?") + " help  " +
		dimStyle.Render("ctrl+c×2") + " quit"

	var tabKeys string
	switch tab {
	case TabFeed:
		tabKeys = dimStyle.Render("j/k") + " scroll  " +
			dimStyle.Render("m") + " merge  " +
			dimStyle.Render("x") + " reject  " +
			dimStyle.Render("a") + " merge all approved"
	case TabQueue:
		tabKeys = dimStyle.Render("j/k") + " navigate"
	case TabBrain:
		tabKeys = dimStyle.Render("j/k") + " navigate  " +
			dimStyle.Render("enter") + " preview  " +
			dimStyle.Render("esc") + " back"
	case TabConfig:
		tabKeys = dimStyle.Render("j/k") + " navigate  " +
			dimStyle.Render("←/→") + " dial"
	}

	result := sectionStyle.Render("Keys") + "\n" + global
	if tabKeys != "" {
		result += "\n" + tabKeys
	}
	return result
}
