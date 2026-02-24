package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/loop"
)

// ReviewsTab renders pending review items and selection state.
type ReviewsTab struct {
	items     []loop.PendingReviewItem
	selection int
}

func (r *ReviewsTab) SetPendingReviews(items []loop.PendingReviewItem) {
	r.items = items
	if r.selection >= len(r.items) {
		r.selection = len(r.items) - 1
	}
	if r.selection < 0 {
		r.selection = 0
	}
}

func (r *ReviewsTab) SelectUp() {
	if r.selection > 0 {
		r.selection--
	}
}

func (r *ReviewsTab) SelectDown() {
	if r.selection < len(r.items)-1 {
		r.selection++
	}
}

func (r *ReviewsTab) SelectedItem() (loop.PendingReviewItem, bool) {
	if r.selection < 0 || r.selection >= len(r.items) {
		return loop.PendingReviewItem{}, false
	}
	return r.items[r.selection], true
}

func (r *ReviewsTab) Render(width, height int) string {
	if len(r.items) == 0 {
		return renderEmptyState("No pending reviews.", width, height)
	}

	var rows []string
	for i, item := range r.items {
		prefix := "  "
		style := dimStyle
		if i == r.selection {
			prefix = "▸ "
			style = successStyle
		}
		line1 := fmt.Sprintf("%s%s  %s  %s", prefix, item.ID, nonEmpty(item.TaskKey, "review"), nonEmpty(item.Model, "model"))
		summary := strings.TrimSpace(item.Title)
		if summary == "" {
			summary = strings.TrimSpace(item.Prompt)
		}
		if summary == "" {
			summary = "(no summary)"
		}
		line2 := "  " + summary
		path := strings.TrimSpace(item.WorktreePath)
		if path == "" {
			path = filepath.Join(".worktrees", item.WorktreeName)
		}
		line3 := dimStyle.Render("  " + path)
		rows = append(rows, style.Render(line1), line2, line3, "")
	}

	footer := dimStyle.Render("enter view diff  m merge  x reject  c request changes")
	rows = append(rows, footer)
	return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
}
