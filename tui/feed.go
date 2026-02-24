package tui

import (
	"fmt"
	"strings"
	"time"
)

// FeedTab manages the feed dashboard view showing one card per agent.
type FeedTab struct {
	agents       []AgentCard
	verdicts     []Verdict
	autonomy     string
	actionNeeded map[string]struct{}
	selection    int
	scroll       int

	// Stats for the footer line.
	activeCount   int
	queuedCount   int
	pendingReview int
	loopState     string
}

// SetSnapshot rebuilds the agent card list from the snapshot's sessions.
// Active agents appear first, then recently completed agents.
func (f *FeedTab) SetSnapshot(snap Snapshot) {
	f.verdicts = snap.Verdicts
	f.autonomy = snap.Autonomy
	f.actionNeeded = make(map[string]struct{}, len(snap.ActionNeeded))
	for _, id := range snap.ActionNeeded {
		f.actionNeeded[id] = struct{}{}
	}

	f.activeCount = len(snap.Active)
	f.queuedCount = len(snap.Queue)
	f.pendingReview = snap.PendingReviewCount
	f.loopState = snap.LoopState

	agents := make([]AgentCard, 0, len(snap.Active)+len(snap.Recent))
	for _, s := range snap.Active {
		agents = append(agents, buildAgentCard(s, snap.EventsBySession[s.ID]))
	}
	for _, s := range snap.Recent {
		agents = append(agents, buildAgentCard(s, snap.EventsBySession[s.ID]))
	}
	f.agents = agents

	// Clamp selection.
	total := f.cardCount()
	if f.selection >= total {
		f.selection = total - 1
	}
	if f.selection < 0 {
		f.selection = 0
	}
}

func buildAgentCard(s Session, events []EventLine) AgentCard {
	card := AgentCard{Session: s}
	if len(events) > 0 {
		last := events[len(events)-1]
		card.LastAction = last.Body
		card.LastLabel = last.Label
	} else if s.CurrentAction != "" {
		card.LastAction = s.CurrentAction
	}
	return card
}

// cardCount returns the total selectable cards (verdicts + agents).
func (f *FeedTab) cardCount() int {
	return len(f.verdicts) + len(f.agents)
}

// SelectDown moves selection to the next card.
func (f *FeedTab) SelectDown() {
	total := f.cardCount()
	if f.selection < total-1 {
		f.selection++
	}
}

// SelectUp moves selection to the previous card.
func (f *FeedTab) SelectUp() {
	if f.selection > 0 {
		f.selection--
	}
}

// SelectedSessionID returns the session ID of the currently selected card.
func (f *FeedTab) SelectedSessionID() string {
	sel := f.selection

	// Verdicts come first.
	if sel < len(f.verdicts) {
		return f.verdicts[sel].SessionID
	}
	sel -= len(f.verdicts)

	if sel >= 0 && sel < len(f.agents) {
		return f.agents[sel].Session.ID
	}
	return ""
}

// Render renders the feed dashboard: verdict banners, agent cards, stats.
func (f *FeedTab) Render(width, height int, now time.Time) string {
	if len(f.agents) == 0 && len(f.verdicts) == 0 {
		return renderEmptyState("No events yet. Warming up the kitchen...", width, height)
	}

	var cards []string
	cardIdx := 0

	// Verdict cards at the top.
	canAct := f.autonomy != "full"
	for _, v := range f.verdicts {
		_, actionable := f.actionNeeded[v.TargetID]
		cards = append(cards, renderVerdictCard(v, width, now, canAct && actionable, cardIdx == f.selection))
		cardIdx++
	}

	// Agent cards: one per session.
	for _, agent := range f.agents {
		cards = append(cards, renderAgentCard(agent, width, now, cardIdx == f.selection))
		cardIdx++
	}

	// Stats line.
	stats := f.renderStats()
	cards = append(cards, stats)

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

func (f *FeedTab) renderStats() string {
	parts := []string{
		fmt.Sprintf("%d active", f.activeCount),
		fmt.Sprintf("%d queued", f.queuedCount),
	}
	if f.pendingReview > 0 {
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%d pending review", f.pendingReview)))
	}
	parts = append(parts, loopStateLabel(f.loopState))
	return "\n" + strings.Join(parts, "  ")
}

// ScrollUp scrolls the feed view up.
func (f *FeedTab) ScrollUp(lines int) {
	f.scroll += lines
}

// ScrollDown scrolls the feed view down.
func (f *FeedTab) ScrollDown(lines int) {
	f.scroll -= lines
	if f.scroll < 0 {
		f.scroll = 0
	}
}
