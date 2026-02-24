package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/tui/components"
)

const railWidth = 21

// renderRail renders the left rail showing agents and stats.
func renderRail(snap Snapshot, height int, shimmerIndex int) string {
	w := railWidth - 4 // border (2) + padding (2)

	var b strings.Builder
	if len(snap.Active) > 0 {
		b.WriteString(sectionStyle.Render("noodle") + " " + renderShimmerTitle("cooking", shimmerIndex))
	} else {
		b.WriteString(sectionStyle.Render("noodle"))
	}
	b.WriteString("\n\n")

	if len(snap.Active) == 0 {
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for _, s := range snap.Active {
			name := nonEmpty(s.DisplayName, s.ID) + runtimeBadge(s.Runtime)
			b.WriteString(healthDot(s.Health) + " " + trimTo(name, w-3))
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
	b.WriteString(loopStateLabel(snap.LoopState))
	b.WriteString("\n")

	content := b.String()
	h := height - 2 // border top + bottom
	if h < 4 {
		h = 4
	}
	return railStyle.Width(railWidth - 2).Height(h).Render(content)
}

// renderCompactRail renders a narrow icon-only rail for terminals < 80 cols.
func renderCompactRail(snap Snapshot, height int) string {
	var b strings.Builder
	for _, s := range snap.Active {
		b.WriteString(healthDot(s.Health))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("─"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%d", len(snap.Active)))
	b.WriteString("\n")

	content := b.String()
	h := height - 2
	if h < 4 {
		h = 4
	}
	return railStyle.Width(6).Height(h).Render(content)
}

// renderShimmerTitle renders the title with a character-by-character yellow shimmer.
func renderShimmerTitle(text string, index int) string {
	t := components.DefaultTheme
	var b strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		if i == index%len(runes) {
			b.WriteString(lipgloss.NewStyle().Foreground(t.Brand).Bold(true).Render(string(r)))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(t.Dim).Render(string(r)))
		}
	}
	return b.String()
}

func shortModelName(model string) string {
	short := map[string]string{
		"claude-opus-4-6":     "opus",
		"claude-sonnet-4-6":   "sonnet",
		"claude-haiku-4-5":    "haiku",
		"gpt-5.3-codex":       "codex",
		"gpt-5.3-codex-spark": "spark",
	}
	if s, ok := short[strings.ToLower(strings.TrimSpace(model))]; ok {
		return s
	}
	return model
}

func runtimeBadge(runtime string) string {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	if runtime == "" || runtime == "tmux" {
		return ""
	}
	return " " + dimStyle.Render(runtime)
}
