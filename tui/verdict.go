package tui

import (
	"fmt"
	"image/color"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/tui/components"
)

// Verdict is a TUI-local view of a quality review result.
type Verdict struct {
	SessionID string
	TargetID  string
	Accept    bool
	Feedback  string
	Timestamp time.Time
}

// loadVerdicts is retained temporarily until the Feed verdict cards are removed.
func loadVerdicts(_ string) []Verdict {
	return nil
}

// renderVerdictCard renders a verdict as a feed card with action pills.
// Optional selected flag highlights the border.
func renderVerdictCard(v Verdict, width int, now time.Time, showActions bool, selected ...bool) string {
	t := components.DefaultTheme
	isSelected := len(selected) > 0 && selected[0]

	var verdictLabel string
	var borderColor color.Color
	if v.Accept {
		verdictLabel = components.VerdictBadge("APPROVE")
		borderColor = t.Success
	} else {
		verdictLabel = components.VerdictBadge("REJECT")
		borderColor = t.Error
	}
	if isSelected {
		borderColor = t.Brand
	}

	targetName := v.TargetID
	if targetName == "" {
		targetName = v.SessionID
	}

	age := components.AgeLabel(now, v.Timestamp)
	ageRendered := lipgloss.NewStyle().Foreground(t.Dim).Render(age)

	title := verdictLabel + " " + targetName
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	titleWidth := lipgloss.Width(title)
	ageWidth := lipgloss.Width(ageRendered)
	gap := innerWidth - titleWidth - ageWidth
	if gap < 1 {
		gap = 1
	}
	titleLine := title + fmt.Sprintf("%*s", gap, "") + ageRendered

	body := v.Feedback
	if body == "" {
		body = "(no feedback)"
	}

	var footer string
	if showActions {
		merge := components.Pill{Label: "Merge", Icon: "m", Color: t.Success, Focused: true}
		reject := components.Pill{Label: "Reject", Icon: "x", Color: t.Error, Focused: false}
		footer = merge.Render(0) + "  " + reject.Render(0)
	}

	card := &components.Card{
		Title:       titleLine,
		Body:        body,
		Footer:      footer,
		BorderColor: borderColor,
	}
	return card.Render(width)
}
