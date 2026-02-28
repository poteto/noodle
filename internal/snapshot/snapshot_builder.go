package snapshot

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/loop"
)

const defaultContextWindowTokens = 200_000

// LoadSnapshot builds a Snapshot from in-memory LoopState. Session data comes
// from the loop's active cook tracking and recent history — no directory
// scanning or bulk event reading.
func LoadSnapshot(runtimeDir string, now time.Time, state loop.LoopState) (Snapshot, error) {
	sessions := make([]Session, 0, len(state.ActiveCooks)+len(state.RecentHistory))
	active := make([]Session, 0, len(state.ActiveCooks))
	recent := make([]Session, 0, len(state.RecentHistory))

	reader := event.NewEventReader(runtimeDir)
	for _, cook := range state.ActiveCooks {
		status := strings.ToLower(strings.TrimSpace(cook.Status))
		if status == "" {
			status = "running"
		}
		session := Session{
			ID:              cook.SessionID,
			DisplayName:     cook.DisplayName,
			Status:          status,
			Runtime:         cook.Runtime,
			Provider:        cook.Provider,
			Model:           cook.Model,
			LastActivity:    now.UTC(),
			TaskKey:         cook.TaskKey,
			TotalCostUSD:    cook.TotalCostUSD,
			DurationSeconds: int64(now.Sub(cook.StartedAt).Seconds()),
		}
		enrichActiveSession(reader, cook.SessionID, &session)
		sessions = append(sessions, session)
		active = append(active, session)
	}

	for _, item := range state.RecentHistory {
		session := Session{
			ID:           item.SessionID,
			DisplayName:  item.SessionID,
			Status:       strings.ToLower(strings.TrimSpace(item.Status)),
			LastActivity: item.CompletedAt,
			TaskKey:      item.TaskKey,
		}
		sessions = append(sessions, session)
		recent = append(recent, session)
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].ID < active[j].ID
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].LastActivity.Equal(recent[j].LastActivity) {
			return recent[i].ID < recent[j].ID
		}
		return recent[i].LastActivity.After(recent[j].LastActivity)
	})

	// Build a lookup from order ID to active cook session ID.
	cookSessionByOrder := make(map[string]string, len(state.ActiveCooks))
	for _, cook := range state.ActiveCooks {
		cookSessionByOrder[cook.OrderID] = cook.SessionID
	}

	orders := make([]Order, 0, len(state.Orders))
	for _, order := range state.Orders {
		activeSessionID := cookSessionByOrder[order.ID]
		stages := make([]Stage, 0, len(order.Stages))
		for _, stage := range order.Stages {
			s := Stage{
				TaskKey:  stage.TaskKey,
				Prompt:   stage.Prompt,
				Skill:    stage.Skill,
				Provider: stage.Provider,
				Model:    stage.Model,
				Runtime:  stage.Runtime,
				Status:   string(stage.Status),
				Extra:    stage.Extra,
			}
			if (stage.Status == orderx.StageStatusActive || stage.Status == orderx.StageStatusMerging) && activeSessionID != "" {
				s.SessionID = activeSessionID
			}
			stages = append(stages, s)
		}
		orders = append(orders, Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    string(order.Status),
		})
	}

	feedEvents := readSteerEvents(filepath.Join(runtimeDir, "control.ndjson"))
	loopEvents := readLoopEvents(runtimeDir)
	feedEvents = append(feedEvents, loopEvents...)
	sort.SliceStable(feedEvents, func(i, j int) bool {
		return feedEvents[i].At.Before(feedEvents[j].At)
	})

	return Snapshot{
		UpdatedAt:          now.UTC(),
		LoopState:          NormalizeLoopState(state.Status),
		Sessions:           sessions,
		Active:             active,
		Recent:             recent,
		Orders:             orders,
		ActiveOrderIDs:     nonNilStrings(cookingOrderIDs(cookSessionByOrder)),
		ActionNeeded:       nonNilStrings(state.ActionNeeded),
		EventsBySession:    map[string][]EventLine{},
		FeedEvents:         nonNilFeedEvents(feedEvents),
		TotalCostUSD:       state.TotalCostUSD,
		PendingReviews:     nonNilReviews(state.PendingReviews),
		PendingReviewCount: state.PendingReviewCount,
		Autonomy:           state.Autonomy,
		MaxCooks:           state.MaxCooks,
	}, nil
}

// enrichActiveSession reads the session's event log and populates
// CurrentAction and ContextWindowUsagePct on the given Session.
func enrichActiveSession(reader *event.EventReader, sessionID string, session *Session) {
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil || len(events) == 0 {
		return
	}

	var totalTokensIn int
	for _, ev := range events {
		switch ev.Type {
		case event.EventAction:
			label, body, _ := formatAction(ev.Payload)
			if body != "" {
				session.CurrentAction = label + " " + body
			}
		case event.EventCost:
			var cost struct {
				TokensIn int `json:"tokens_in"`
			}
			if json.Unmarshal(ev.Payload, &cost) == nil {
				totalTokensIn += cost.TokensIn
			}
		}
	}

	pct := float64(totalTokensIn) / defaultContextWindowTokens * 100
	if pct > 100 {
		pct = 100
	}
	session.ContextWindowUsagePct = pct
}

// NormalizeLoopState maps a raw loop state string to a canonical constant.
func NormalizeLoopState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "idle":
		return LoopStateIdle
	case "running":
		return LoopStateRunning
	case "paused":
		return LoopStatePaused
	case "draining":
		return LoopStateDraining
	default:
		return ""
	}
}

// cookingOrderIDs returns order IDs that have an active cook session.
// This is a subset of the loop's ActiveOrderIDs — it excludes orders that are
// "active" but waiting to be dispatched (no cook yet).
func cookingOrderIDs(sessionByOrder map[string]string) []string {
	ids := make([]string, 0, len(sessionByOrder))
	for orderID := range sessionByOrder {
		ids = append(ids, orderID)
	}
	sort.Strings(ids)
	return ids
}

// nonNil helpers ensure slices marshal to [] instead of null in JSON.

func nonNilStrings(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return append([]string(nil), s...)
}

func nonNilFeedEvents(s []FeedEvent) []FeedEvent {
	if s == nil {
		return []FeedEvent{}
	}
	return s
}

func nonNilReviews(s []loop.PendingReviewItem) []loop.PendingReviewItem {
	if s == nil {
		return []loop.PendingReviewItem{}
	}
	return append([]loop.PendingReviewItem(nil), s...)
}

