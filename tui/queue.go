package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/poteto/noodle/tui/components"
)

// QueueStatus describes where a queue item is in the lifecycle.
type QueueStatus string

const (
	QueueStatusCooking   QueueStatus = "cooking"
	QueueStatusReviewing QueueStatus = "reviewing"
	QueueStatusPlanned   QueueStatus = "planned"
	QueueStatusNoPlan    QueueStatus = "no plan"
	QueueStatusReady     QueueStatus = "ready"
)

// QueueTab renders the queue as a table with progress bar.
type QueueTab struct {
	table table.Model
	items []QueueItem
	stats queueStats
}

type queueStats struct {
	total  int
	cooked int
}

// noPlanTaskTypes are task types that don't require a plan.
var noPlanTaskTypes = map[string]struct{}{
	"reflect":    {},
	"prioritize": {},
	"quality":    {},
}

// NewQueueTab creates a fresh QueueTab.
func NewQueueTab() QueueTab {
	cols := []table.Column{
		{Title: "#", Width: 3},
		{Title: "Type", Width: 10},
		{Title: "Item", Width: 40},
		{Title: "Status", Width: 10},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(queueTableStyles())
	return QueueTab{table: t}
}

func queueTableStyles() table.Styles {
	t := components.DefaultTheme
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Foreground(t.Brand).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(t.Dim)
	s.Cell = lipgloss.NewStyle().
		Foreground(t.Text)
	s.Selected = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(lipgloss.Color("#2a2a40")).
		Bold(true)
	return s
}

// SetQueue updates the table data from snapshot.
func (q *QueueTab) SetQueue(items []QueueItem, activeIDs []string, actionNeeded []string) {
	q.items = items
	activeSet := make(map[string]struct{}, len(activeIDs))
	for _, id := range activeIDs {
		activeSet[id] = struct{}{}
	}
	actionSet := make(map[string]struct{}, len(actionNeeded))
	for _, id := range actionNeeded {
		actionSet[id] = struct{}{}
	}

	cooked := 0
	rows := make([]table.Row, 0, len(items))
	for i, item := range items {
		status := deriveQueueStatus(item, activeSet, actionSet)
		if status == QueueStatusCooking || status == QueueStatusReviewing {
			cooked++
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			item.TaskKey,
			truncTitle(item.Title, item.ID),
			string(status),
		})
	}
	q.table.SetRows(rows)
	q.stats = queueStats{total: len(items), cooked: cooked}
}

// Render draws the queue tab content.
func (q *QueueTab) Render(width, height int) string {
	q.resizeTable(width, height)

	var b strings.Builder
	b.WriteString(q.renderProgressLine(width))
	b.WriteString("\n\n")
	b.WriteString(q.renderStyledTable())
	return b.String()
}

func (q *QueueTab) renderProgressLine(width int) string {
	t := components.DefaultTheme
	cooked := q.stats.cooked
	total := q.stats.total
	label := fmt.Sprintf("%d/%d cooked  ", cooked, total)
	labelWidth := len(label)
	barWidth := width - labelWidth - 2
	if barWidth < 10 {
		barWidth = 10
	}
	labelStyle := lipgloss.NewStyle().Foreground(t.Secondary)
	return labelStyle.Render(label) + components.ProgressBar(float64(cooked), float64(total), barWidth)
}

func (q *QueueTab) resizeTable(width, height int) {
	// Reserve space for progress bar (1 line) + blank line (1 line)
	tableHeight := height - 2
	if tableHeight < 3 {
		tableHeight = 3
	}
	q.table.SetHeight(tableHeight)

	// Distribute column widths: #(3), Type(10), Status(10), Item(rest)
	numW := 3
	typeW := 10
	statusW := 10
	// Subtract column gaps (3 gaps * 2 chars each for padding)
	itemW := width - numW - typeW - statusW - 6
	if itemW < 10 {
		itemW = 10
	}
	q.table.SetColumns([]table.Column{
		{Title: "#", Width: numW},
		{Title: "Type", Width: typeW},
		{Title: "Item", Width: itemW},
		{Title: "Status", Width: statusW},
	})
}

// renderStyledTable renders the table view with colored Type and Status columns.
func (q *QueueTab) renderStyledTable() string {
	raw := q.table.View()
	if len(q.items) == 0 {
		return dimStyle.Render("(queue empty)")
	}
	return raw
}

// deriveQueueStatus determines the status of a queue item.
func deriveQueueStatus(item QueueItem, activeIDs map[string]struct{}, actionNeeded map[string]struct{}) QueueStatus {
	// Cooking: currently running in an active session
	if _, ok := activeIDs[item.ID]; ok {
		return QueueStatusCooking
	}

	// Reviewing: in action_needed list (completed, verdict exists, awaiting merge)
	if _, ok := actionNeeded[item.ID]; ok {
		return QueueStatusReviewing
	}

	// Task types that don't need plans are "ready"
	taskKey := strings.ToLower(strings.TrimSpace(item.TaskKey))
	if _, ok := noPlanTaskTypes[taskKey]; ok {
		return QueueStatusReady
	}

	// Execute tasks: check if review exists (has a plan) or not
	if item.Review != nil {
		return QueueStatusPlanned
	}

	return QueueStatusNoPlan
}

func truncTitle(title, id string) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	return id
}

// StatusStyle returns the appropriate style for a queue status.
func statusStyleForQueue(status QueueStatus) lipgloss.Style {
	t := components.DefaultTheme
	switch status {
	case QueueStatusCooking:
		return lipgloss.NewStyle().Foreground(t.Success)
	case QueueStatusReviewing:
		return lipgloss.NewStyle().Foreground(t.Info)
	case QueueStatusPlanned:
		return lipgloss.NewStyle().Foreground(t.Plan)
	case QueueStatusNoPlan:
		return lipgloss.NewStyle().Foreground(t.Warning)
	case QueueStatusReady:
		return lipgloss.NewStyle().Foreground(t.Success)
	default:
		return lipgloss.NewStyle().Foreground(t.Dim)
	}
}
