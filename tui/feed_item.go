package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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

// FeedItem groups consecutive events from the same session into a card.
type FeedItem struct {
	SessionID string
	AgentName string
	TaskType  string
	Category  string
	Events    []FeedEvent
	StartedAt time.Time

	// Steer metadata (only when Category == "steer").
	SteerTarget string
	SteerPrompt string
}

// renderFeedItem renders a single feed item as a Card component.
func renderFeedItem(item FeedItem, width int, now time.Time) string {
	t := components.DefaultTheme

	var title string
	var borderColor lipgloss.Color

	if item.Category == "steer" {
		borderColor = t.Brand
		targetName := item.SteerTarget
		if targetName == "" && item.AgentName != "" {
			targetName = item.AgentName
		}
		title = fmt.Sprintf("★ Chef → %s", targetName)
	} else {
		borderColor = taskTypeBorderColor(item.TaskType)
		badge := components.TaskTypeBadge(item.TaskType)
		title = badge + " " + item.AgentName
	}

	age := components.AgeLabel(now, item.StartedAt)
	ageRendered := lipgloss.NewStyle().Foreground(t.Dim).Render(age)

	// Right-align timestamp: pad title to fill available space.
	innerWidth := width - 4 // subtract card border + padding
	if innerWidth < 20 {
		innerWidth = 20
	}
	titleWidth := lipgloss.Width(title)
	ageWidth := lipgloss.Width(ageRendered)
	gap := innerWidth - titleWidth - ageWidth
	if gap < 1 {
		gap = 1
	}
	titleLine := title + strings.Repeat(" ", gap) + ageRendered

	dimBody := lipgloss.NewStyle().Foreground(t.Dim)
	var body strings.Builder
	for i, ev := range item.Events {
		if i > 0 {
			body.WriteString("\n")
		}
		if item.Category == "steer" {
			body.WriteString(fmt.Sprintf("%q", ev.Body))
		} else {
			label := eventLabel(ev.Label)
			bodyText := dimBody.Render(components.TrimTo(ev.Body, innerWidth-lipgloss.Width(ev.Label)-2))
			body.WriteString(label + " " + bodyText)
		}
	}

	card := &components.Card{
		Title:       titleLine,
		Body:        body.String(),
		BorderColor: borderColor,
	}
	return card.Render(width)
}

func taskTypeBorderColor(taskType string) lipgloss.Color {
	t := components.DefaultTheme
	switch taskType {
	case "execute", "Execute":
		return t.Execute
	case "plan", "Plan":
		return t.Plan
	case "quality", "Quality":
		return t.Quality
	case "reflect", "Reflect":
		return t.Reflect
	case "prioritize", "Prioritize":
		return t.Prioritize
	default:
		return t.Border
	}
}
