package loop

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/stringx"
)

func (l *Loop) controlMode(value string) error {
	value = stringx.Normalize(value)
	switch value {
	case string(state.RunModeAuto), string(state.RunModeSupervised), string(state.RunModeManual):
		l.config.Mode = value

		// Emit V2 canonical state event for mode change.
		l.emitEvent(ingest.EventModeChanged, map[string]any{
			"mode":         value,
			"requested_by": "control",
			"reason":       "mode control command",
		})

		return nil
	default:
		return fmt.Errorf("unsupported mode value %q", value)
	}
}

func (l *Loop) controlEnqueue(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("enqueue requires order_id")
	}
	prompt := strings.TrimSpace(cmd.Prompt)
	taskKey := strings.TrimSpace(cmd.TaskKey)
	if taskKey == "" {
		taskKey = "execute"
	}

	newOrder := Order{
		ID:     orderID,
		Title:  titleFromPrompt(prompt, 8),
		Status: OrderStatusActive,
		Stages: []Stage{{
			TaskKey:  taskKey,
			Prompt:   prompt,
			Skill:    strings.TrimSpace(cmd.Skill),
			Provider: strings.TrimSpace(cmd.Provider),
			Model:    strings.TrimSpace(cmd.Model),
			Status:   StageStatusPending,
		}},
	}
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		orders.Orders = append(orders.Orders, newOrder)
		return true, nil
	}); err != nil {
		return err
	}

	// Emit V2 canonical state event for new order promotion.
	stages := make([]map[string]any, len(newOrder.Stages))
	for i, s := range newOrder.Stages {
		stages[i] = map[string]any{
			"stage_index": i,
			"status":      "pending",
			"skill":       s.Skill,
			"runtime":     s.Runtime,
		}
	}
	l.emitEvent(ingest.EventSchedulePromoted, map[string]any{
		"order_id": orderID,
		"stages":   stages,
	})

	return nil
}

func (l *Loop) controlEditItem(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("edit-item requires order_id")
	}
	if _, active := l.cooks.activeCooksByOrder[orderID]; active {
		return fmt.Errorf("order %q is currently cooking", orderID)
	}
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			changed := false

			// Edit order-level fields.
			if title := strings.TrimSpace(cmd.Prompt); title != "" {
				nextTitle := titleFromPrompt(title, 8)
				if orders.Orders[i].Title != nextTitle {
					orders.Orders[i].Title = nextTitle
					changed = true
				}
			}

			// Edit stage-level fields on the current pending stage.
			stageIdx, stage := activeStageForOrder(orders.Orders[i])
			if stageIdx < 0 || stage == nil {
				return false, fmt.Errorf("order %q has no editable stage", orderID)
			}
			if prompt := strings.TrimSpace(cmd.Prompt); prompt != "" && orders.Orders[i].Stages[stageIdx].Prompt != prompt {
				orders.Orders[i].Stages[stageIdx].Prompt = prompt
				changed = true
			}
			if taskKey := strings.TrimSpace(cmd.TaskKey); taskKey != "" && orders.Orders[i].Stages[stageIdx].TaskKey != taskKey {
				orders.Orders[i].Stages[stageIdx].TaskKey = taskKey
				changed = true
			}
			if provider := strings.TrimSpace(cmd.Provider); provider != "" && orders.Orders[i].Stages[stageIdx].Provider != provider {
				orders.Orders[i].Stages[stageIdx].Provider = provider
				changed = true
			}
			if model := strings.TrimSpace(cmd.Model); model != "" && orders.Orders[i].Stages[stageIdx].Model != model {
				orders.Orders[i].Stages[stageIdx].Model = model
				changed = true
			}
			if skill := strings.TrimSpace(cmd.Skill); skill != "" && orders.Orders[i].Stages[stageIdx].Skill != skill {
				orders.Orders[i].Stages[stageIdx].Skill = skill
				changed = true
			}
			return changed, nil
		}
		return false, fmt.Errorf("order %q not found", orderID)
	})
}

func (l *Loop) controlSkip(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("skip requires order_id")
	}
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, err := cancelOrder(*orders, orderID)
		if err != nil {
			return false, err
		}
		*orders = updated
		return true, nil
	})
}

func (l *Loop) controlRequeue(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("requeue requires order_id")
	}

	// Reset failed/cancelled stages to pending and reactivate the order.
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != orderID {
				continue
			}
			wasFailed := orders.Orders[i].Status == OrderStatusFailed
			orders.Orders[i].Status = OrderStatusActive
			changed := resetStages(&orders.Orders[i].Stages)
			updated := changed || wasFailed
			if !updated {
				return false, fmt.Errorf("order %q not in failed state", orderID)
			}
			return true, nil
		}
		return false, fmt.Errorf("order %q not found", orderID)
	}); err != nil {
		return err
	}
	_ = l.events.Emit(LoopEventOrderRequeued, OrderRequeuedPayload{
		OrderID: orderID,
	})
	return nil
}

// resetStages resets all failed/cancelled stages to pending and reports whether
// any stage changed.
func resetStages(stages *[]Stage) bool {
	changed := false
	for i := range *stages {
		switch (*stages)[i].Status {
		case StageStatusFailed, StageStatusCancelled:
			(*stages)[i].Status = StageStatusPending
			changed = true
		}
	}
	return changed
}

func (l *Loop) controlReorder(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("reorder requires order_id")
	}
	newIndex := 0
	if v := strings.TrimSpace(cmd.Value); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("reorder: invalid index %q", v)
		}
		newIndex = n
	}
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		srcIdx := -1
		for i := range orders.Orders {
			if orders.Orders[i].ID == orderID {
				srcIdx = i
				break
			}
		}
		if srcIdx < 0 {
			return false, fmt.Errorf("order %q not found", orderID)
		}
		targetIdx := newIndex
		if targetIdx < 0 {
			targetIdx = 0
		}
		if targetIdx >= len(orders.Orders) {
			targetIdx = len(orders.Orders) - 1
		}
		if srcIdx == targetIdx {
			return false, nil
		}
		order := orders.Orders[srcIdx]
		orders.Orders = append(orders.Orders[:srcIdx], orders.Orders[srcIdx+1:]...)
		orders.Orders = append(orders.Orders[:targetIdx], append([]Order{order}, orders.Orders[targetIdx:]...)...)
		return true, nil
	})
}

func (l *Loop) controlSetMaxConcurrency(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("set-max-concurrency requires value")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("set-max-concurrency: invalid value %q", value)
	}
	if n < 1 {
		return fmt.Errorf("max_concurrency must be at least 1")
	}
	l.config.Concurrency.MaxConcurrency = n
	return nil
}
