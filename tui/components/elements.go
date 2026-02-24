package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/internal/stringx"
)

// HealthDot returns a colored dot (●) based on health status.
func HealthDot(health string) string {
	t := DefaultTheme
	dot := "●"
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "red":
		return lipgloss.NewStyle().Foreground(t.Error).Render(dot)
	case "yellow":
		return lipgloss.NewStyle().Foreground(t.Warning).Render(dot)
	default:
		return lipgloss.NewStyle().Foreground(t.Success).Render(dot)
	}
}

// StatusIcon returns a semantic icon for session status.
func StatusIcon(status string) string {
	t := DefaultTheme
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return lipgloss.NewStyle().Foreground(t.Error).Render("x")
	case "completed", "exited":
		return lipgloss.NewStyle().Foreground(t.Success).Render("✓")
	default:
		return lipgloss.NewStyle().Foreground(t.Dim).Render("~")
	}
}

// AgeLabel returns a human-readable age string (e.g., "3m", "1h").
func AgeLabel(now, at time.Time) string {
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

// DurationLabel returns a formatted duration string.
func DurationLabel(seconds int64) string {
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

// CostLabel returns a formatted cost string.
func CostLabel(usd float64) string {
	t := DefaultTheme
	return lipgloss.NewStyle().Foreground(t.Brand).Render(fmt.Sprintf("$%.2f", usd))
}

// TrimTo truncates a string using middle truncation for readability.
func TrimTo(value string, width int) string {
	return stringx.MiddleTruncate(strings.TrimSpace(value), width)
}
