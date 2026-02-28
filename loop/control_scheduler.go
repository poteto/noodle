package loop

import (
	"context"
	"fmt"
	"strings"
)

// controlAdvance advances past a stage blocked by a stage_message.
// Constructs a synthetic cookHandle and calls advanceAndPersist for full
// completion side effects (events, adapter "done" on final stage).
func (l *Loop) controlAdvance(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("advance requires order_id")
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

	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	found := false
	for i := range orders.Orders {
		if orders.Orders[i].ID != orderID {
			continue
		}
		found = true
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
		// Ensure order is active so the new stage can be dispatched.
		if orders.Orders[i].Status == OrderStatusFailed {
			orders.Orders[i].Status = OrderStatusActive
		}
		break
	}
	if !found {
		return fmt.Errorf("order %q not found", orderID)
	}
	return l.writeOrdersState(orders)
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

	l.cooks.pendingReview[orderID] = &pendingReviewCook{
		cookIdentity: cookIdentity{
			orderID:    orderID,
			stageIndex: stageIdx,
			stage:      *stage,
			plan:       order.Plan,
		},
		worktreeName: worktreeName,
		worktreePath: worktreePath,
		sessionID:    sessionID,
		reason:       reason,
	}
	return l.writePendingReview()
}
