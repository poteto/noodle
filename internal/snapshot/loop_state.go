package snapshot

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/loop"
)

func LoadSnapshotFromLoopState(runtimeDir string, now time.Time, state loop.LoopState) (Snapshot, error) {
	sessions := make([]Session, 0, len(state.ActiveCooks)+len(state.RecentHistory))
	active := make([]Session, 0, len(state.ActiveCooks))
	recent := make([]Session, 0, len(state.RecentHistory))

	for _, cook := range state.ActiveCooks {
		status := strings.ToLower(strings.TrimSpace(cook.Status))
		if status == "" {
			status = "running"
		}
		session := Session{
			ID:           cook.SessionID,
			DisplayName:  cook.DisplayName,
			Status:       status,
			Runtime:      cook.Runtime,
			Provider:     cook.Provider,
			Model:        cook.Model,
			LastActivity: now.UTC(),
			TaskKey:      cook.TaskKey,
		}
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

	orders := make([]Order, 0, len(state.Orders))
	for _, order := range state.Orders {
		stages := make([]Stage, 0, len(order.Stages))
		for _, stage := range order.Stages {
			stages = append(stages, Stage{
				TaskKey:  stage.TaskKey,
				Prompt:   stage.Prompt,
				Skill:    stage.Skill,
				Provider: stage.Provider,
				Model:    stage.Model,
				Runtime:  stage.Runtime,
				Status:   stage.Status,
				Extra:    stage.Extra,
			})
		}
		onFailure := make([]Stage, 0, len(order.OnFailure))
		for _, stage := range order.OnFailure {
			onFailure = append(onFailure, Stage{
				TaskKey:  stage.TaskKey,
				Prompt:   stage.Prompt,
				Skill:    stage.Skill,
				Provider: stage.Provider,
				Model:    stage.Model,
				Runtime:  stage.Runtime,
				Status:   stage.Status,
				Extra:    stage.Extra,
			})
		}
		orders = append(orders, Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    order.Status,
			OnFailure: onFailure,
		})
	}

	feedEvents := readSteerEvents(filepath.Join(runtimeDir, "control.ndjson"))
	queueEvents := readQueueEvents(runtimeDir)
	feedEvents = append(feedEvents, queueEvents...)
	sort.SliceStable(feedEvents, func(i, j int) bool {
		return feedEvents[i].At.Before(feedEvents[j].At)
	})

	return Snapshot{
		UpdatedAt:          now.UTC(),
		LoopState:          normalizeLoopState(state.Status),
		Sessions:           sessions,
		Active:             active,
		Recent:             recent,
		Orders:             orders,
		ActiveOrderIDs:     append([]string(nil), state.ActiveOrderIDs...),
		ActionNeeded:       append([]string(nil), state.ActionNeeded...),
		EventsBySession:    map[string][]EventLine{},
		FeedEvents:         feedEvents,
		TotalCostUSD:       state.TotalCostUSD,
		PendingReviews:     append([]loop.PendingReviewItem(nil), state.PendingReviews...),
		PendingReviewCount: state.PendingReviewCount,
		Autonomy:           state.Autonomy,
		MaxCooks:           state.MaxCooks,
	}, nil
}

func ReadSessionEvents(runtimeDir, sessionID string) ([]EventLine, error) {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil {
		if os.IsNotExist(err) {
			return []EventLine{}, nil
		}
		return nil, err
	}
	return mapEventLines(events), nil
}
