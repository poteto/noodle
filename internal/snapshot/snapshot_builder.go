package snapshot

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/projection"
	"github.com/poteto/noodle/internal/stringx"
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
	projected := structuralSnapshotView(state)

	reader := event.NewEventReader(runtimeDir)
	for _, cook := range state.ActiveCooks {
		status := stringx.Normalize(cook.Status)
		if status == "" {
			status = "running"
		}
		runtime := stringx.Normalize(cook.Runtime)
		var remoteHost string
		if runtime != "" && runtime != "process" {
			remoteHost = runtime
		}
		session := Session{
			ID:              cook.SessionID,
			DisplayName:     cook.DisplayName,
			Status:          status,
			Runtime:         cook.Runtime,
			Provider:        cook.Provider,
			Model:           cook.Model,
			LastActivity:    now.UTC(),
			WorktreeName:    cook.WorktreeName,
			TaskKey:         cook.TaskKey,
			TotalCostUSD:    cook.TotalCostUSD,
			DurationSeconds: int64(now.Sub(cook.StartedAt).Seconds()),
			RemoteHost:      remoteHost,
		}
		enrichActiveSession(reader, cook.SessionID, &session)
		sessions = append(sessions, session)
		active = append(active, session)
	}

	for _, item := range state.RecentHistory {
		session := Session{
			ID:           item.SessionID,
			DisplayName:  item.SessionID,
			Status:       stringx.Normalize(item.Status),
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

	// Build a lookup from (order ID, task key) to completed session ID so
	// stages that finished keep a navigable session_id in the UI.
	type orderStageKey struct{ orderID, taskKey string }
	completedSessionByStage := make(map[orderStageKey]string, len(state.RecentHistory))
	for _, item := range state.RecentHistory {
		if item.OrderID != "" && item.TaskKey != "" && item.SessionID != "" {
			completedSessionByStage[orderStageKey{item.OrderID, item.TaskKey}] = item.SessionID
		}
	}

	orders := make([]Order, 0, len(projected.Orders))
	for _, order := range projected.Orders {
		activeSessionID := cookSessionByOrder[order.ID]
		stages := make([]Stage, 0, len(order.Stages))
		for _, stage := range order.Stages {
			s := Stage{
				TaskKey:   stage.TaskKey,
				Prompt:    stage.Prompt,
				Skill:     stage.Skill,
				Provider:  stage.Provider,
				Model:     stage.Model,
				Runtime:   stage.Runtime,
				Group:     stage.Group,
				Status:    stage.Status,
				Extra:     cloneRawMap(stage.Extra),
				SessionID: "",
			}
			if (stage.Status == string(orderx.StageStatusActive) || stage.Status == string(orderx.StageStatusMerging)) && activeSessionID != "" {
				s.SessionID = activeSessionID
			} else if sid := completedSessionByStage[orderStageKey{order.ID, s.TaskKey}]; sid != "" {
				s.SessionID = sid
			}
			stages = append(stages, s)
		}
		orders = append(orders, Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    order.Status,
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
		ActionNeeded:       nonNilStrings(projected.ActionNeeded),
		EventsBySession:    map[string][]EventLine{},
		FeedEvents:         nonNilFeedEvents(feedEvents),
		TotalCostUSD:       state.TotalCostUSD,
		PendingReviews:     projectedPendingReviews(projected.PendingReviews),
		PendingReviewCount: projected.PendingReviewCount,
		Mode:               projected.Mode,
		MaxConcurrency:     state.MaxConcurrency,
	}, nil
}

func structuralSnapshotView(state loop.LoopState) projection.SnapshotView {
	if len(state.Projection.Orders) != 0 || len(state.Projection.PendingReviews) != 0 || state.Projection.SchemaVersion != 0 {
		return state.Projection
	}

	orders := make([]projection.OrderProjection, 0, len(state.Orders))
	activeOrderIDs := make([]string, 0, len(state.Orders))
	for _, order := range state.Orders {
		orders = append(orders, projection.OrderProjection{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Status:    string(order.Status),
			Stages:    projectLoopStages(order.Stages),
		})
		if order.Status != orderx.OrderStatusCompleted && order.Status != orderx.OrderStatusFailed {
			activeOrderIDs = append(activeOrderIDs, order.ID)
		}
	}
	sort.Strings(activeOrderIDs)

	return projection.SnapshotView{
		Orders:             orders,
		ActiveOrderIDs:     activeOrderIDs,
		ActionNeeded:       append([]string(nil), state.ActionNeeded...),
		PendingReviews:     projectLoopPendingReviews(state.PendingReviews),
		PendingReviewCount: state.PendingReviewCount,
		Mode:               state.Mode,
		ModeEpoch:          state.ModeEpoch,
		GeneratedAt:        state.UpdatedAt,
	}
}

func projectLoopStages(stages []loop.Stage) []projection.StageProjection {
	projected := make([]projection.StageProjection, 0, len(stages))
	for _, stage := range stages {
		projected = append(projected, projection.StageProjection{
			TaskKey:     stage.TaskKey,
			Prompt:      stage.Prompt,
			Skill:       stage.Skill,
			Provider:    stage.Provider,
			Model:       stage.Model,
			Runtime:     stage.Runtime,
			Group:       stage.Group,
			Status:      string(stage.Status),
			Extra:       cloneRawMap(stage.Extra),
			ExtraPrompt: stage.ExtraPrompt,
		})
	}
	return projected
}

func projectLoopPendingReviews(reviews []loop.PendingReviewItem) []projection.PendingReviewProjection {
	projected := make([]projection.PendingReviewProjection, 0, len(reviews))
	for _, review := range reviews {
		projected = append(projected, projection.PendingReviewProjection{
			OrderID:      review.OrderID,
			StageIndex:   review.StageIndex,
			TaskKey:      review.TaskKey,
			Prompt:       review.Prompt,
			Provider:     review.Provider,
			Model:        review.Model,
			Runtime:      review.Runtime,
			Skill:        review.Skill,
			Plan:         append([]string(nil), review.Plan...),
			WorktreeName: review.WorktreeName,
			WorktreePath: review.WorktreePath,
			SessionID:    review.SessionID,
			Reason:       review.Reason,
		})
	}
	return projected
}

func projectedPendingReviews(reviews []projection.PendingReviewProjection) []loop.PendingReviewItem {
	items := make([]loop.PendingReviewItem, 0, len(reviews))
	for _, review := range reviews {
		items = append(items, loop.PendingReviewItem{
			OrderID:      review.OrderID,
			StageIndex:   review.StageIndex,
			TaskKey:      review.TaskKey,
			Prompt:       review.Prompt,
			Provider:     review.Provider,
			Model:        review.Model,
			Runtime:      review.Runtime,
			Skill:        review.Skill,
			Plan:         append([]string(nil), review.Plan...),
			WorktreeName: review.WorktreeName,
			WorktreePath: review.WorktreePath,
			SessionID:    review.SessionID,
			Reason:       review.Reason,
		})
	}
	return items
}

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
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
			// Cost events are emitted from canonical result events.
			// Treat them as turn completion so stale prior action text does not
			// keep the UI in a perpetual "thinking" state after interrupts.
			session.CurrentAction = ""
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
	switch stringx.Normalize(value) {
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
