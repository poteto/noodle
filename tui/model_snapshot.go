package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/snapshot"
	"github.com/poteto/noodle/internal/stringx"
)

func isActiveStatus(status string) bool { return snapshot.IsActiveStatus(status) }
func inferTaskType(sessionID string) string { return snapshot.InferTaskType(sessionID) }

func healthDot(health string) string {
	dot := "●"
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "red":
		return errorStyle.Render(dot)
	case "yellow":
		return warnStyle.Render(dot)
	default:
		return successStyle.Render(dot)
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return errorStyle.Render("x")
	case "completed", "exited":
		return successStyle.Render("ok")
	default:
		return dimStyle.Render("~")
	}
}

func modelLabel(session Session) string {
	switch {
	case session.Provider != "" && session.Model != "":
		return session.Provider + "/" + session.Model
	case session.Model != "":
		return session.Model
	case session.Provider != "":
		return session.Provider
	default:
		return "-"
	}
}

func ageLabel(now, at time.Time) string {
	if at.IsZero() {
		return "-"
	}
	if now.Before(at) {
		return "0s"
	}
	seconds := int64(now.Sub(at).Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	return fmt.Sprintf("%dh", hours)
}

func durationLabel(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func trimTo(value string, width int) string {
	return stringx.MiddleTruncate(strings.TrimSpace(value), width)
}

func padRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return value + strings.Repeat(" ", width-len(runes))
}

func wrapPlainText(value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{""}
	}
	if width <= 1 {
		return []string{value}
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	paragraphs := strings.Split(value, "\n")
	out := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapParagraph(paragraph, width)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapParagraph(paragraph string, width int) []string {
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		for runeLen(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			head, tail := splitAtRuneWidth(word, width)
			lines = append(lines, head)
			word = tail
		}
		if current == "" {
			current = word
			continue
		}
		candidate := current + " " + word
		if runeLen(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func runeLen(value string) int {
	return len([]rune(value))
}

func splitAtRuneWidth(value string, width int) (head string, tail string) {
	if width <= 0 {
		return "", value
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value, ""
	}
	return string(runes[:width]), string(runes[width:])
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return timestamp.UTC()
}

func nonEmpty(value, fallback string) string {
	return stringx.NonEmpty(value, fallback)
}
