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
	Category  string // "steer", "ticket", "action", "cost", "queue_drop", "registry_rebuild", "bootstrap", etc.
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
	name := s.DisplayName
	if name == "" {
		name = s.ID
	}

	var titlePrefix string
	if selected {
		borderColor = t.Brand
		titlePrefix = "▸ "
	}

	var icon string
	if done {
		icon = components.StatusIcon(s.Status)
	} else {
		icon = healthDot(s.Health)
	}

	badge := components.TaskTypeBadge(inferTaskType(s.ID))
	model := shortModelName(nonEmpty(s.Model, "-"))

	titleParts := icon + " " + name +
		" " + dimStyle.Render("──") + " " + badge +
		" " + dimStyle.Render("──") + " " + dimStyle.Render(model)
	if done {
		titleParts += " " + dimStyle.Render("── done")
	}
	title := titlePrefix + titleParts

	innerWidth := width - 4 // card border + padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Body line 1: last action.
	var bodyLines []string
	if card.LastAction != "" {
		label := eventLabel(card.LastLabel)
		maxBody := innerWidth - lipgloss.Width(card.LastLabel) - 2
		if maxBody < 10 {
			maxBody = 10
		}
		actionText := components.TrimTo(card.LastAction, maxBody)
		if done {
			bodyLines = append(bodyLines, dimStyle.Render(card.LastLabel+" "+actionText))
		} else {
			bodyLines = append(bodyLines, label+" "+lipgloss.NewStyle().Foreground(t.Dim).Render(actionText))
		}
	} else {
		bodyLines = append(bodyLines, dimStyle.Render("(waiting...)"))
	}

	// Body line 2: context bar + duration + cost.
	dur := durationLabel(s.DurationSeconds)
	var metaParts []string
	barWidth := 24
	if s.ContextWindowUsagePct > 0 {
		metaParts = append(metaParts, components.ProgressBar(s.ContextWindowUsagePct, 1.0, barWidth))
	} else {
		metaParts = append(metaParts, components.ProgressBar(0, 1.0, barWidth))
	}
	metaParts = append(metaParts, dimStyle.Render(dur))
	if s.TotalCostUSD > 0 {
		metaParts = append(metaParts, costStyle.Render(fmt.Sprintf("$%.2f", s.TotalCostUSD)))
	}
	bodyLines = append(bodyLines, strings.Join(metaParts, "  "))

	c := &components.Card{
		Title:       title,
		Body:        strings.Join(bodyLines, "\n"),
		BorderColor: borderColor,
	}
	return c.Render(width)
}

func taskTypeBorderColor(taskType string) color.Color {
	return components.TaskTypeColor(taskType)
}
