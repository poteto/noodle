package loop

import (
	"context"
	"fmt"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
)

// controlAdvance force-completes the active stage of an order.
// Kills any running session, removes it from the active cook map, then
// constructs a synthetic cookHandle and calls advanceAndPersist for full
// completion side effects (events, adapter "done" on final stage).
func (l *Loop) controlAdvance(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("advance requires order_id")
	}

	// Kill the active session to prevent double-processing when it exits.
	if cook, ok := l.cooks.activeCooksByOrder[orderID]; ok {
		if err := cook.session.ForceKill(); err != nil {
			return fmt.Errorf("force kill active session for order %q failed: %w", orderID, err)
		}
		l.trackCookCompleted(cook, StageResult{
			SessionID:   cook.session.ID(),
			Status:      StageResultCancelled,
			CompletedAt: l.deps.Now(),
		})
		delete(l.cooks.activeCooksByOrder, orderID)
	}

	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	for _, o := range orders.Orders {
		if o.ID != orderID {
			continue
		}
		stageIdx, stage := activeStageForOrder(o)
		if stageIdx < 0 || stage == nil {
			return fmt.Errorf("order %q has no advanceable stage", orderID)
		}
		cook := &cookHandle{
			cookIdentity: cookIdentity{
				orderID:    orderID,
				stageIndex: stageIdx,
				stage:      *stage,
			},
			orderStatus: o.Status,
			session:     &adoptedSession{id: "", status: "completed"},
		}
		if err := l.ensureCanonicalOrderFromOrders(orderID); err != nil {
			return err
		}
		if err := l.emitEventChecked(ingest.EventStageCompleted, map[string]any{
			"order_id":    orderID,
			"stage_index": stageIdx,
			"mergeable":   false,
		}); err != nil {
			return err
		}
		return l.advanceAndPersist(context.Background(), cook)
	}
	return fmt.Errorf("order %q not found", orderID)
}

// controlAddStage inserts a new stage into an existing order's pipeline.
// The stage is inserted after the last completed or failed stage.
func (l *Loop) controlAddStage(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("add-stage requires order_id")
	}
	taskKey := strings.TrimSpace(cmd.TaskKey)
	if taskKey == "" {
		return fmt.Errorf("add-stage requires task_key")
	}

	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			changed := false
			newStage := Stage{
				TaskKey:  taskKey,
				Prompt:   strings.TrimSpace(cmd.Prompt),
				Skill:    strings.TrimSpace(cmd.Skill),
				Provider: strings.TrimSpace(cmd.Provider),
				Model:    strings.TrimSpace(cmd.Model),
				Status:   StageStatusPending,
			}
			// Insert after the last completed or failed stage.
			insertAt := 0
			for j, s := range orders.Orders[i].Stages {
				if s.Status == StageStatusCompleted || s.Status == StageStatusFailed {
					insertAt = j + 1
				}
			}
			stages := orders.Orders[i].Stages
			stages = append(stages[:insertAt], append([]Stage{newStage}, stages[insertAt:]...)...)
			orders.Orders[i].Stages = stages
			changed = true
			// Ensure order is active so the new stage can be dispatched.
			if orders.Orders[i].Status == OrderStatusFailed {
				orders.Orders[i].Status = OrderStatusActive
			}
			return changed, nil
		}
		return false, fmt.Errorf("order %q not found", orderID)
	})
}

// controlParkReview parks an order for human review. The scheduler uses this
// as the sole creation path for pending reviews.
func (l *Loop) controlParkReview(orderID, reason string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("park-review requires order_id")
	}
	reason = strings.TrimSpace(reason)

	// Look up order to get stage info.
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	var order *Order
	for i := range orders.Orders {
		if orders.Orders[i].ID == orderID {
			order = &orders.Orders[i]
			break
		}
	}
	if order == nil {
		return fmt.Errorf("order %q not found", orderID)
	}

	// Determine stage info — use the active/most-recent stage.
	stageIdx, stage := activeStageForOrder(*order)
	if stageIdx < 0 {
		// Fall back to last stage if none are active/pending.
		if len(order.Stages) > 0 {
			stageIdx = len(order.Stages) - 1
			stage = &order.Stages[stageIdx]
		} else {
			return fmt.Errorf("order %q has no stages", orderID)
		}
	}

	// Check if there's an active cook — use its session/worktree info.
	var sessionID, worktreeName, worktreePath string
	if cook, ok := l.cooks.activeCooksByOrder[orderID]; ok {
		sessionID = cook.session.ID()
		worktreeName = cook.worktreeName
		worktreePath = cook.worktreePath
	}
	if err := l.ensureCanonicalOrderFromOrders(orderID); err != nil {
		return err
	}

	if err := l.emitEventChecked(ingest.EventStageReviewParked, map[string]any{
		"order_id":      orderID,
		"stage_index":   stageIdx,
		"session_id":    sessionID,
		"worktree_name": worktreeName,
		"worktree_path": worktreePath,
		"reason":        reason,
		"task_key":      stage.TaskKey,
		"prompt":        stage.Prompt,
		"provider":      stage.Provider,
		"model":         stage.Model,
		"runtime":       stage.Runtime,
		"skill":         stage.Skill,
		"plan":          append([]string(nil), order.Plan...),
	}); err != nil {
		return err
	}
	if err := l.mirrorLegacyOrderFromCanonical(orderID); err != nil {
		return err
	}
	return l.syncPendingReviewProjection()
}
