package tui

// renderLayout composes the split layout: left rail + tabbed right pane.
func (m Model) renderLayout() string {
	if m.width <= 0 || m.height <= 0 {
		return titleStyle.Render("noodle") + " " + dimStyle.Render("loading...")
	}

	paneWidth := m.width - railWidth - 1
	if paneWidth < 20 {
		paneWidth = 20
	}

	rail := renderRail(m.snapshot, m.now(), m.height)
	tabBar := renderTabBar(m.activeTab, paneWidth)

	var tabContent string
	switch m.activeTab {
	case TabFeed:
		tabContent = dimStyle.Render("(feed — coming soon)")
	case TabQueue:
		tabContent = dimStyle.Render("(queue — coming soon)")
	case TabBrain:
		tabContent = dimStyle.Render("(brain — coming soon)")
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

	return layout
}

func renderHelp() string {
	return sectionStyle.Render("Keys") + "\n" +
		dimStyle.Render("1-4") + " switch tabs  " +
		dimStyle.Render("s") + " steer  " +
		dimStyle.Render("p") + " pause/resume  " +
		dimStyle.Render("?") + " help  " +
		dimStyle.Render("ctrl+c") + " quit"
}
