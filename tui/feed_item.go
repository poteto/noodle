package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/tui/components"
)

// FeedEvent is one event in the feed timeline.
type FeedEvent struct {
	SessionID string
	AgentName string
	TaskType  string
	At        time.Time
	Label     string
	Body      string
	Category  string // "steer", "ticket", "action", "cost", etc.
}

// AgentCard represents one agent's current state in the feed dashboard.
type AgentCard struct {
	Session    Session
	LastAction string // most recent event body
	LastLabel  string // most recent event label
}

// renderAgentCard renders a single agent's dashboard card.
func renderAgentCard(card AgentCard, width int, now time.Time, selected bool) string {
	t := components.DefaultTheme
	s := card.Session

	done := !isActiveStatus(s.Status)
	borderColor := taskTypeBorderColor(inferTaskType(s.ID))
	if done {
		borderColor = t.Dim
	}

	// Title: health dot + name + task type + model
	var title string
	name := s.DisplayName
	if name == "" {
		name = s.ID
	}
	if selected {
		borderColor = t.Brand
		title = "▸ "
	}

	if done {
		title += components.StatusIcon(s.Status)
	} else {
		title += healthDot(s.Health)
	}
	title += " " + name

	badge := components.TaskTypeBadge(inferTaskType(s.ID))
	model := shortModelName(nonEmpty(s.Model, "-"))
	title += " " + dimStyle.Render("──") + " " + badge + " " + dimStyle.Render("──") + " " + dimStyle.Render(model)

	if done {
		title += " " + dimStyle.Render("── done")
	}

	innerWidth := width - 4 // card border + padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Body: last action (single line).
	var body string
	if card.LastAction != "" {
		label := eventLabel(card.LastLabel)
		actionText := components.TrimTo(card.LastAction, innerWidth-lipgloss.Width(card.LastLabel)-2)
		body = label + " " + lipgloss.NewStyle().Foreground(t.Dim).Render(actionText)
	} else {
		body = dimStyle.Render("(waiting...)")
	}

	// Footer: context bar + duration + cost
	dur := durationLabel(s.DurationSeconds)
	var footerParts []string
	if s.ContextWindowUsagePct > 0 {
		barWidth := 24
		footerParts = append(footerParts, components.ProgressBar(s.ContextWindowUsagePct, 1.0, barWidth))
	}
	footerParts = append(footerParts, dimStyle.Render(dur))
	if s.TotalCostUSD > 0 {
		footerParts = append(footerParts, costStyle.Render(fmt.Sprintf("$%.2f", s.TotalCostUSD)))
	}
	footer := strings.Join(footerParts, "  ")

	c := &components.Card{
		Title:       title,
		Body:        body,
		Footer:      footer,
		BorderColor: borderColor,
	}
	return c.Render(width)
}

func taskTypeBorderColor(taskType string) color.Color {
	return components.TaskTypeColor(taskType)
}
