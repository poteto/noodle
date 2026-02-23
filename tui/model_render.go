package tui

import (
	"fmt"
	"strings"
	"unicode"
)

func (m Model) renderSurface(surface Surface) string {
	switch surface {
	case surfaceSession:
		return m.renderSession()
	case surfaceTrace:
		return m.renderTrace()
	case surfaceQueue:
		return m.renderQueue()
	default:
		return m.renderDashboard()
	}
}

func (m Model) renderDashboard() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("noodle"))
	b.WriteString(dimStyle.Render(" | "))
	b.WriteString(accentStyle.Render("cooking"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s %s | %s %d | %s %d | %s %s",
		labelStyle.Render("status"),
		loopStateLabel(m.snapshot.LoopState),
		labelStyle.Render("active"),
		len(m.snapshot.Active),
		labelStyle.Render("queue"),
		len(m.snapshot.Queue),
		labelStyle.Render("total"),
		costStyle.Render(fmt.Sprintf("$%.2f", m.snapshot.TotalCostUSD)),
	)
	if !m.snapshot.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, " | updated %s ago", mutedStyle.Render(ageLabel(m.now(), m.snapshot.UpdatedAt)))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	b.WriteString(sectionLine("Active Cooks", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Active) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for i, session := range m.snapshot.Active {
			cursor := " "
			if i == m.selectedActive {
				cursor = accentStyle.Render(">")
			}
			action := nonEmpty(session.CurrentAction, "(idle)")
			line := fmt.Sprintf(
				"%s %s %-22s %-20s %-26s %6s ago\n",
				cursor,
				healthDot(session.Health),
				session.ID,
				modelLabel(session),
				trimTo(action, 22),
				ageLabel(m.now(), session.LastActivity),
			)
			if i == m.selectedActive {
				line = selectedRowStyle.Render(trimTo(strings.TrimSuffix(line, "\n"), bodyWidth))
				b.WriteString(line)
				b.WriteString("\n")
				continue
			}
			b.WriteString(line)
		}
	}

	b.WriteString("\nRecent\n")
	b.WriteString(sectionLine("Recent", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Recent) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		limit := 6
		if len(m.snapshot.Recent) < limit {
			limit = len(m.snapshot.Recent)
		}
		for i := 0; i < limit; i++ {
			s := m.snapshot.Recent[i]
			fmt.Fprintf(
				&b,
				"  %s %-22s %-20s %8s %s\n",
				statusIcon(s.Status)+" "+statusLabel(s.Status),
				s.ID,
				modelLabel(s),
				durationLabel(s.DurationSeconds),
				costStyle.Render(fmt.Sprintf("$%.2f", s.TotalCostUSD)),
			)
		}
	}

	b.WriteString("\n")
	b.WriteString(sectionLine("Up Next", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(empty)"))
		b.WriteString("\n")
	} else {
		limit := 6
		if len(m.snapshot.Queue) < limit {
			limit = len(m.snapshot.Queue)
		}
		for i := 0; i < limit; i++ {
			item := m.snapshot.Queue[i]
			titleWidth := bodyWidth - 56
			if titleWidth < 20 {
				titleWidth = 20
			}
			fmt.Fprintf(
				&b,
				"  %d. %-16s %-12s %-12s %s\n",
				i+1,
				trimTo(item.ID, 16),
				infoStyle.Render(nonEmpty(item.Provider, "claude")),
				trimTo(queueTaskDisplayName(item), 12),
				trimTo(nonEmpty(strings.TrimSpace(item.Title), "(untitled)"), titleWidth),
			)
		}
	}

	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("enter inspect | q queue | s steer | p pause/resume | d drain | ? help | ctrl+c quit"))
	return b.String()
}

func (m Model) renderSession() string {
	bodyWidth := m.contentWidth()
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return errorStyle.Render("session not found") + "\n\n" + keybarStyle.Render("esc back")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s\n", titleStyle.Render("Session Detail"), accentStyle.Render(session.ID))
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")

	fmt.Fprintf(&b, "%s %s %s\n", labelStyle.Render("Status:"), statusLabel(session.Status), healthDot(session.Health))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Provider:"), nonEmpty(session.Provider, "-"))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Model:"), nonEmpty(session.Model, "-"))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Duration:"), durationLabel(session.DurationSeconds))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Cost:"), costStyle.Render(fmt.Sprintf("$%.2f", session.TotalCostUSD)))
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Retries:"), session.RetryCount)
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Worktree:"), mutedStyle.Render(".worktrees/"+session.ID))

	lines := m.traceDisplayLines(m.snapshot.EventsBySession[session.ID])
	height := m.sessionEventsHeight()
	start := 0
	if len(lines) > height {
		if m.sessionEventsFollow {
			start = len(lines) - height
		} else {
			start = m.sessionEventsOffset
			maxStart := len(lines) - height
			if start < 0 {
				start = 0
			}
			if start > maxStart {
				start = maxStart
			}
		}
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	b.WriteString("\n")
	b.WriteString(sectionLine("Recent Events", bodyWidth))
	b.WriteString("\n")
	if len(lines) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for _, line := range lines[start:end] {
			atCell := padRight(strings.TrimSpace(line.At), 8)
			labelCell := padRight(strings.TrimSpace(line.Label), 14)
			if strings.TrimSpace(labelCell) == "" {
				fmt.Fprintf(&b, "  %s  %s | %s\n", atCell, labelCell, line.Body)
				continue
			}
			fmt.Fprintf(&b, "  %s  %s | %s\n", atCell, eventLabel(labelCell), line.Body)
		}
	}

	b.WriteString("\n")
	if m.sessionEventsFollow {
		b.WriteString(infoStyle.Render("[events auto-scroll]\n"))
	}
	b.WriteString(keybarStyle.Render("t trace | up/down events | k kill | s steer | esc back | ? help"))
	return b.String()
}

func (m Model) renderTrace() string {
	bodyWidth := m.contentWidth()
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return errorStyle.Render("trace unavailable: session not found") + "\n\n" + keybarStyle.Render("esc back")
	}
	lines := m.traceDisplayLines(m.filteredTraceLines())
	height := m.traceHeight()
	start := 0
	if len(lines) > height {
		if m.traceFollow {
			start = len(lines) - height
		} else {
			start = m.traceOffset
			maxStart := len(lines) - height
			if start < 0 {
				start = 0
			}
			if start > maxStart {
				start = maxStart
			}
		}
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s | %s %s\n",
		titleStyle.Render("Trace"),
		accentStyle.Render(session.ID),
		mutedStyle.Render("filter:"),
		infoStyle.Render(string(m.traceFilter)),
	)
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("(no events)\n"))
	} else {
		for _, line := range lines[start:end] {
			atCell := padRight(strings.TrimSpace(line.At), 8)
			labelCell := padRight(strings.TrimSpace(line.Label), 14)
			if strings.TrimSpace(labelCell) == "" {
				fmt.Fprintf(&b, "%s  %s | %s\n", atCell, labelCell, line.Body)
				continue
			}
			fmt.Fprintf(&b, "%s  %s | %s\n", atCell, eventLabel(labelCell), line.Body)
		}
	}
	if m.traceFollow {
		b.WriteString("\n")
		b.WriteString(infoStyle.Render("[auto-scroll]"))
	}
	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("f filter | G bottom | esc back | ? help"))
	return b.String()
}

func (m Model) renderQueue() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Queue\n"))
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString(dimStyle.Render("(empty)\n"))
	} else {
		for i, item := range m.snapshot.Queue {
			cursor := " "
			if i == m.selectedQueue {
				cursor = accentStyle.Render(">")
			}
			review := "default"
			if item.Review != nil {
				if *item.Review {
					review = "review"
				} else {
					review = "no-review"
				}
			}
			titleWidth := bodyWidth - 70
			if titleWidth < 20 {
				titleWidth = 20
			}
			line := fmt.Sprintf(
				"%s %2d. %-20s %-12s %-18s %-9s %s\n",
				cursor,
				i+1,
				trimTo(item.ID, 20),
				infoStyle.Render(nonEmpty(item.Provider, "-")),
				trimTo(nonEmpty(item.Model, "-"), 18),
				review,
				trimTo(nonEmpty(strings.TrimSpace(item.Title), "(untitled)"), titleWidth),
			)
			if i == m.selectedQueue {
				line = selectedRowStyle.Render(trimTo(strings.TrimSuffix(line, "\n"), bodyWidth))
				b.WriteString(line)
				b.WriteString("\n")
				continue
			}
			b.WriteString(line)
		}
	}
	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("esc back | s steer | ? help"))
	return b.String()
}

func renderHelp() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("Keys"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 24)))
	b.WriteString("\n")
	b.WriteString("Global: s steer | p pause/resume | d drain | ? help | ctrl+c quit\n")
	b.WriteString("Dashboard: enter inspect | q queue | up/down move\n")
	b.WriteString("Session: t trace | up/down scroll events | k kill | esc back\n")
	b.WriteString("Trace: f filter | G bottom | up/down scroll | esc back\n")
	b.WriteString("Steer: type @target + instruction; @everyone for broadcast")
	return b.String()
}

func queueTaskDisplayName(item QueueItem) string {
	token := strings.TrimSpace(item.TaskKey)
	if token == "" {
		token = strings.TrimSpace(item.Skill)
	}
	if token == "" {
		id := strings.TrimSpace(item.ID)
		if isNumeric(id) {
			return "Execute"
		}
		if head, _, ok := strings.Cut(id, "-"); ok && strings.TrimSpace(head) != "" {
			token = head
		}
	}
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "prioritize":
		return "Prioritize"
	default:
		return titleCaseToken(token, "Task")
	}
}

func titleCaseToken(token string, fallback string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	token = strings.ReplaceAll(token, "_", " ")
	token = strings.ReplaceAll(token, "-", " ")
	parts := strings.Fields(token)
	if len(parts) == 0 {
		return fallback
	}
	for i := range parts {
		runes := []rune(parts[i])
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func isNumeric(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (m Model) renderSteer() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Steer"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Instruction"))
	b.WriteString("\n")
	if strings.TrimSpace(m.steerInput) == "" {
		b.WriteString(dimStyle.Render("> @cook-a focus on tests, keep commits small"))
	} else {
		b.WriteString("> ")
		b.WriteString(m.steerInput)
	}

	if m.steerMentionOpen && len(m.steerMentionItems) > 0 {
		b.WriteString("\n\n")
		b.WriteString(sectionLine("Mentions", bodyWidth))
		b.WriteString("\n")
		for i, mention := range m.steerMentionItems {
			row := "  " + mention
			if i == m.steerMentionIndex {
				row = selectedRowStyle.Render(trimTo(strings.TrimSpace(row), bodyWidth))
			}
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Type @ to mention cooks. Enter submits. Esc closes mentions, then closes steer."))
	return b.String()
}
