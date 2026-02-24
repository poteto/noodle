package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	table     table.Model
	items     []QueueItem
	stats     queueStats
	loopState string
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
		{Title: "Type", Width: 12},
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
	// Selected style matches Cell — full-row highlight is applied in renderStyledTable.
	s.Selected = s.Cell
	return s
}

// SetQueue updates the table data from snapshot.
func (q *QueueTab) SetQueue(items []QueueItem, activeIDs []string, actionNeeded []string, loopState string) {
	q.items = items
	q.loopState = loopState
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
	b.WriteString(q.renderStyledTable(width, height-2))
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

	// Distribute column widths: #(3), Type(12), Status(10), Item(rest)
	numW := 3
	typeW := 12
	statusW := 10
	// Subtract column gaps (3 gaps * 3 chars each for padding)
	itemW := width - numW - typeW - statusW - 9
	if itemW < 10 {
		itemW = 10
	}
	q.table.SetColumns([]table.Column{
		{Title: "#  ", Width: numW},
		{Title: "Type", Width: typeW},
		{Title: "Item", Width: itemW},
		{Title: "Status", Width: statusW},
	})
}

// renderStyledTable renders the table view with colored Type and Status columns.
// Post-processes the selected row to apply background color across the full width.
func (q *QueueTab) renderStyledTable(width, height int) string {
	if len(q.items) == 0 {
		msg := "Queue is empty. A prioritize agent will fill it\nfrom your backlog shortly."
		if q.loopState == "idle" {
			msg = "All plans complete. Start a new conversation to create more."
		}
		return renderEmptyState(msg, width, height)
	}
	raw := q.table.View()
	lines := strings.Split(raw, "\n")

	// The table renders: header (1 line) + border (1 line) + data rows.
	// The cursor row in data is at offset 2 + cursor.
	cursor := q.table.Cursor()
	selectedLine := 2 + cursor
	if cursor >= 0 && selectedLine < len(lines) {
		t := components.DefaultTheme
		// Strip existing ANSI codes so inner cell styles don't override the background.
		stripped := ansi.Strip(lines[selectedLine])
		bg := lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a40")).
			Foreground(t.Brand).
			Bold(true).
			Width(width)
		lines[selectedLine] = bg.Render(stripped)
	}
	return strings.Join(lines, "\n")
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

	// Items with linked plans are ready to execute.
	if len(item.Plan) > 0 {
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

// SelectedSessionID returns the queue item ID of the currently selected row.
// This is used as a prefix match against session IDs to open the actor view.
func (q *QueueTab) SelectedSessionID() string {
	cursor := q.table.Cursor()
	if cursor < 0 || cursor >= len(q.items) {
		return ""
	}
	return q.items[cursor].ID
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
