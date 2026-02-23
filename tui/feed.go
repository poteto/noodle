package tui

import (
	"strings"
	"time"
)

const maxFeedItems = 100

// FeedTab manages the feed view state.
type FeedTab struct {
	items        []FeedItem
	verdicts     []Verdict
	autonomy     string
	actionNeeded map[string]struct{}
	selection    int
	scroll       int
	userScroll   bool // true when user has scrolled up manually
}

// SetSnapshot rebuilds feed items from the snapshot's FeedEvents.
// Consecutive events from the same session are grouped into one card.
func (f *FeedTab) SetSnapshot(snap Snapshot) {
	f.verdicts = snap.Verdicts
	f.autonomy = snap.Autonomy
	f.actionNeeded = make(map[string]struct{}, len(snap.ActionNeeded))
	for _, id := range snap.ActionNeeded {
		f.actionNeeded[id] = struct{}{}
	}
	events := snap.FeedEvents
	if len(events) == 0 {
		f.items = nil
		return
	}

	items := make([]FeedItem, 0, len(events))
	var current *FeedItem

	for _, ev := range events {
		// Group consecutive events from the same session into one card.
		if current != nil && current.SessionID == ev.SessionID {
			current.Events = append(current.Events, ev)
			continue
		}
		// Flush current group.
		if current != nil {
			items = append(items, *current)
		}
		item := FeedItem{
			SessionID: ev.SessionID,
			AgentName: ev.AgentName,
			TaskType:  ev.TaskType,
			Category:  ev.Category,
			Events:    []FeedEvent{ev},
			StartedAt: ev.At,
		}
		if ev.Category == "steer" {
			item.SteerTarget = ev.AgentName
			item.SteerPrompt = ev.Body
		}
		current = &item
	}
	if current != nil {
		items = append(items, *current)
	}

	// Cap items.
	if len(items) > maxFeedItems {
		items = items[len(items)-maxFeedItems:]
	}

	f.items = items

	// Auto-scroll to newest (top) unless user has scrolled away.
	if !f.userScroll {
		f.scroll = 0
	}
}

// Render renders the feed tab content for the given dimensions.
// Items are displayed newest first (reverse chronological).
func (f *FeedTab) Render(width, height int, now time.Time) string {
	if len(f.items) == 0 && len(f.verdicts) == 0 {
		return dimStyle.Render("No events yet. Waiting for activity...")
	}

	var cards []string

	// Verdict cards appear at the top (most actionable).
	canAct := f.autonomy != "full"
	for _, v := range f.verdicts {
		_, actionable := f.actionNeeded[v.TargetID]
		cards = append(cards, renderVerdictCard(v, width, now, canAct && actionable))
	}

	// Show a limited number of recent cards based on terminal height.
	// Each card is ~4 lines (border + title + events + spacing).
	maxCards := height / 4
	if maxCards < 3 {
		maxCards = 3
	}
	if maxCards > 8 {
		maxCards = 8
	}

	// Render items in reverse-chronological order (newest first).
	shown := 0
	for i := len(f.items) - 1; i >= 0 && shown < maxCards; i-- {
		card := renderFeedItem(f.items[i], width, now)
		cards = append(cards, card)
		shown++
	}

	all := strings.Join(cards, "\n")
	allLines := strings.Split(all, "\n")

	// Apply scroll offset.
	start := f.scroll
	if start < 0 {
		start = 0
	}
	if start >= len(allLines) {
		start = len(allLines) - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + height
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]
	return strings.Join(visible, "\n")
}

// ScrollUp scrolls the feed view up (toward older content).
// Disengages auto-scroll.
func (f *FeedTab) ScrollUp(lines int) {
	f.scroll += lines
	f.userScroll = true
}

// ScrollDown scrolls the feed view down (toward newest content).
// Re-engages auto-scroll if back at top.
func (f *FeedTab) ScrollDown(lines int) {
	f.scroll -= lines
	if f.scroll <= 0 {
		f.scroll = 0
		f.userScroll = false
	}
}
