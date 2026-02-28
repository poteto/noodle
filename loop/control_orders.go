package loop

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/poteto/noodle/internal/state"
)

func (l *Loop) controlMode(value string) error {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case string(state.RunModeAuto), string(state.RunModeSupervised), string(state.RunModeManual):
		l.config.Mode = value
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

	orders, err := l.currentOrders()
	if err != nil {
		return err
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
	orders.Orders = append(orders.Orders, newOrder)
	return l.writeOrdersState(orders)
}

func (l *Loop) controlEditItem(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("edit-item requires order_id")
	}
	if _, active := l.cooks.activeCooksByOrder[orderID]; active {
		return fmt.Errorf("order %q is currently cooking", orderID)
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
		// Edit order-level fields.
		if title := strings.TrimSpace(cmd.Prompt); title != "" {
			orders.Orders[i].Title = titleFromPrompt(title, 8)
		}
		// Edit stage-level fields on the current pending stage.
		stageIdx, stage := activeStageForOrder(orders.Orders[i])
		if stageIdx < 0 || stage == nil {
			return fmt.Errorf("order %q has no editable stage", orderID)
		}
		if prompt := strings.TrimSpace(cmd.Prompt); prompt != "" {
			orders.Orders[i].Stages[stageIdx].Prompt = prompt
		}
		if taskKey := strings.TrimSpace(cmd.TaskKey); taskKey != "" {
			orders.Orders[i].Stages[stageIdx].TaskKey = taskKey
		}
		if provider := strings.TrimSpace(cmd.Provider); provider != "" {
			orders.Orders[i].Stages[stageIdx].Provider = provider
		}
		if model := strings.TrimSpace(cmd.Model); model != "" {
			orders.Orders[i].Stages[stageIdx].Model = model
		}
		if skill := strings.TrimSpace(cmd.Skill); skill != "" {
			orders.Orders[i].Stages[stageIdx].Skill = skill
		}
		break
	}
	if !found {
		return fmt.Errorf("order %q not found", orderID)
	}
	return l.writeOrdersState(orders)
}

func (l *Loop) controlSkip(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("skip requires order_id")
	}
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, err = cancelOrder(orders, orderID)
	if err != nil {
		return err
	}
	return l.writeOrdersState(orders)
}

func (l *Loop) controlRequeue(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("requeue requires order_id")
	}
	if _, ok := l.cooks.failedTargets[orderID]; !ok {
		return fmt.Errorf("order %q not in failed state", orderID)
	}

	// If order still exists in orders.json, reset all failed/cancelled stages
	// to "pending", set Order.Status to "active".
	// Write orders BEFORE mutating in-memory failedTargets to avoid divergence
	// on I/O errors.
	orders, err := l.currentOrders()
	if err != nil {
		return fmt.Errorf("requeue: read orders: %w", err)
	}
	updated := false
	for i := range orders.Orders {
		if orders.Orders[i].ID != orderID {
			continue
		}
		orders.Orders[i].Status = OrderStatusActive
		resetStages(&orders.Orders[i].Stages)
		updated = true
		break
	}
	if updated {
		if err := l.writeOrdersState(orders); err != nil {
			return err
		}
	}
	_ = l.events.Emit(LoopEventOrderRequeued, OrderRequeuedPayload{
		OrderID: orderID,
	})
	delete(l.cooks.failedTargets, orderID)
	return l.writeFailedTargets()
}

// resetStages resets all failed/cancelled stages to pending.
func resetStages(stages *[]Stage) {
	for i := range *stages {
		switch (*stages)[i].Status {
		case StageStatusFailed, StageStatusCancelled:
			(*stages)[i].Status = StageStatusPending
		}
	}
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
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	srcIdx := -1
	for i := range orders.Orders {
		if orders.Orders[i].ID == orderID {
			srcIdx = i
			break
		}
	}
	if srcIdx < 0 {
		return fmt.Errorf("order %q not found", orderID)
	}
	order := orders.Orders[srcIdx]
	orders.Orders = append(orders.Orders[:srcIdx], orders.Orders[srcIdx+1:]...)
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex > len(orders.Orders) {
		newIndex = len(orders.Orders)
	}
	orders.Orders = append(orders.Orders[:newIndex], append([]Order{order}, orders.Orders[newIndex:]...)...)
	return l.writeOrdersState(orders)
}

func (l *Loop) controlSetMaxCooks(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("set-max-cooks requires value")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("set-max-cooks: invalid value %q", value)
	}
	if n < 1 {
		return fmt.Errorf("max_cooks must be at least 1")
	}
	l.config.Concurrency.MaxCooks = n
	return nil
}
