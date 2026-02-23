package tui

import (
	"fmt"
	"strings"
	"time"
)

const railWidth = 21

// renderRail renders the left rail showing agents and stats.
func renderRail(snap Snapshot, now time.Time, height int) string {
	w := railWidth - 4 // border (2) + padding (2)

	var b strings.Builder
	b.WriteString(sectionStyle.Render("Agents"))
	b.WriteString("\n\n")

	if len(snap.Active) == 0 && len(snap.Recent) == 0 {
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for _, s := range snap.Active {
			b.WriteString(healthDot(s.Health) + " " + trimTo(s.ID, w-3))
			b.WriteString("\n")
			model := shortModelName(nonEmpty(s.Model, "-"))
			dur := durationLabel(s.DurationSeconds)
			b.WriteString("  " + dimStyle.Render(model+" · "+dur))
			b.WriteString("\n")
		}
		limit := 2
		if len(snap.Recent) < limit {
			limit = len(snap.Recent)
		}
		for i := 0; i < limit; i++ {
			s := snap.Recent[i]
			b.WriteString(successStyle.Render("✓") + " " + trimTo(s.ID, w-3))
			b.WriteString("\n")
			dur := durationLabel(s.DurationSeconds)
			b.WriteString("  " + dimStyle.Render("done · "+dur))
			b.WriteString("\n")
		}
	}

	b.WriteString(dimStyle.Render(strings.Repeat("─", w)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%d active\n", len(snap.Active)))
	b.WriteString(fmt.Sprintf("%d queued\n", len(snap.Queue)))
	if snap.PendingReviewCount > 0 {
		b.WriteString(warnStyle.Render(fmt.Sprintf("%d pending review", snap.PendingReviewCount)))
		b.WriteString("\n")
	}
	b.WriteString(costStyle.Render(fmt.Sprintf("$%.2f", snap.TotalCostUSD)))
	b.WriteString("\n")

	content := b.String()
	h := height - 2 // border top + bottom
	if h < 4 {
		h = 4
	}
	return railStyle.Width(railWidth - 2).Height(h).Render(content)
}

func shortModelName(model string) string {
	short := map[string]string{
		"claude-opus-4-6":      "opus",
		"claude-sonnet-4-6":    "sonnet",
		"claude-haiku-4-5":     "haiku",
		"gpt-5.3-codex":       "codex",
		"gpt-5.3-codex-spark": "spark",
	}
	if s, ok := short[strings.ToLower(strings.TrimSpace(model))]; ok {
		return s
	}
	return model
}
