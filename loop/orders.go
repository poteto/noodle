package loop

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
)

func activeTicketTargetSet(brief mise.Brief) map[string]struct{} {
	targets := make(map[string]struct{}, len(brief.Tickets))
	for _, ticket := range brief.Tickets {
		target := strings.TrimSpace(ticket.Target)
		if target == "" {
			continue
		}
		switch ticket.Status {
		case "active", "blocked":
			targets[target] = struct{}{}
		}
	}
	return targets
}

// dispatchCandidate is a lightweight struct identifying a stage ready for dispatch.
type dispatchCandidate struct {
	OrderID     string
	StageIndex  int
	Stage       Stage
	IsOnFailure bool
}

// activeStageForOrder returns the index and pointer to the currently active or
// first pending stage. Returns (-1, nil) if no stage is active/pending.
func activeStageForOrder(order Order) (int, *Stage) {
	stages := order.Stages
	if order.Status == OrderStatusFailing {
		stages = order.OnFailure
	}
	for i := range stages {
		switch stages[i].Status {
		case StageStatusActive, StageStatusPending:
			return i, &stages[i]
		}
	}
	return -1, nil
}

// advanceOrder marks the current active/first-pending stage as completed.
// For "active" orders: if all main stages complete, removes the order and returns removed=true.
// For "failing" orders: advances through OnFailure stages; last one completing removes the order.
func advanceOrder(orders OrdersFile, orderID string) (OrdersFile, bool, error) {
	idx := -1
	for i := range orders.Orders {
		if orders.Orders[i].ID == orderID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return orders, false, fmt.Errorf("order %q not found", orderID)
	}

	orders = cloneOrdersFile(orders)
	order := &orders.Orders[idx]

	stages := &order.Stages
	if order.Status == OrderStatusFailing {
		stages = &order.OnFailure
	}

	// Find and complete the current active/first-pending stage.
	advanced := false
	for i := range *stages {
		switch (*stages)[i].Status {
		case StageStatusActive, StageStatusPending:
			(*stages)[i].Status = StageStatusCompleted
			advanced = true
		}
		if advanced {
			break
		}
	}
	if !advanced {
		return orders, false, fmt.Errorf("order %q has no active or pending stage to advance", orderID)
	}

	// Check if all stages in the relevant pipeline are completed.
	allDone := true
	for _, s := range *stages {
		if s.Status != StageStatusCompleted {
			allDone = false
			break
		}
	}

	if allDone {
		orders.Orders = slices.Delete(orders.Orders, idx, idx+1)
		return orders, true, nil
	}

	return orders, false, nil
}

// failStage marks the current active stage as failed and handles the failure pipeline.
// If the order has OnFailure stages and is not already "failing": cancels remaining main
// stages, sets order to "failing", resets OnFailure stages to "pending".
// If no OnFailure or already failing: cancels remaining stages and removes the order.
// Returns terminal=true when the order is removed (caller calls markFailed).
func failStage(orders OrdersFile, orderID string, reason string) (OrdersFile, bool, error) {
	idx := -1
	for i := range orders.Orders {
		if orders.Orders[i].ID == orderID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return orders, false, fmt.Errorf("order %q not found", orderID)
	}

	orders = cloneOrdersFile(orders)
	order := &orders.Orders[idx]

	// Determine which pipeline we're operating on.
	if order.Status == OrderStatusFailing {
		// Already in failure pipeline — fail the current OnFailure stage, cancel rest, remove.
		failCurrentAndCancelRest(&order.OnFailure)
		orders.Orders = slices.Delete(orders.Orders, idx, idx+1)
		return orders, true, nil
	}

	// Mark current main stage as failed, cancel remaining main stages.
	failCurrentAndCancelRest(&order.Stages)

	// If OnFailure stages exist, transition to "failing".
	if len(order.OnFailure) > 0 {
		order.Status = OrderStatusFailing
		for i := range order.OnFailure {
			order.OnFailure[i].Status = StageStatusPending
		}
		return orders, false, nil
	}

	// No OnFailure — terminal removal.
	orders.Orders = slices.Delete(orders.Orders, idx, idx+1)
	return orders, true, nil
}

// failCurrentAndCancelRest marks the first active/pending stage as failed
// and all subsequent non-completed stages as cancelled.
func failCurrentAndCancelRest(stages *[]Stage) {
	foundCurrent := false
	for i := range *stages {
		s := &(*stages)[i]
		if !foundCurrent {
			if s.Status == StageStatusActive || s.Status == StageStatusPending {
				s.Status = StageStatusFailed
				foundCurrent = true
			}
		} else {
			if s.Status != StageStatusCompleted {
				s.Status = StageStatusCancelled
			}
		}
	}
}

// cancelOrder marks all non-completed stages as cancelled and removes the order.
func cancelOrder(orders OrdersFile, orderID string) (OrdersFile, error) {
	idx := -1
	for i := range orders.Orders {
		if orders.Orders[i].ID == orderID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return orders, fmt.Errorf("order %q not found", orderID)
	}

	orders = cloneOrdersFile(orders)
	order := &orders.Orders[idx]

	for i := range order.Stages {
		if order.Stages[i].Status != StageStatusCompleted {
			order.Stages[i].Status = StageStatusCancelled
		}
	}
	for i := range order.OnFailure {
		if order.OnFailure[i].Status != StageStatusCompleted {
			order.OnFailure[i].Status = StageStatusCancelled
		}
	}

	orders.Orders = slices.Delete(orders.Orders, idx, idx+1)
	return orders, nil
}

// ActiveStageOrderIDs returns the IDs of orders that have at least one stage
// with status "active". Used for busy-set derivation: an active stage means a
// cook was dispatched for that order (written to disk before spawn), so the
// order should be blocked from re-dispatch even if the in-memory map is empty
// (e.g. after crash/restart).
func ActiveStageOrderIDs(orders OrdersFile) map[string]struct{} {
	result := make(map[string]struct{})
	for _, order := range orders.Orders {
		stages := order.Stages
		if order.Status == OrderStatusFailing {
			stages = order.OnFailure
		}
		for _, s := range stages {
			if s.Status == StageStatusActive {
				result[order.ID] = struct{}{}
				break
			}
		}
	}
	return result
}

// dispatchableStages finds the first pending stage per order that is ready for dispatch.
// Orders in busy/adopted/ticketed sets are skipped. Orders in the failed set are skipped
// unless they are in "failing" status (OnFailure must dispatch).
func dispatchableStages(orders OrdersFile, busy, failed, adopted, ticketed map[string]struct{}) []dispatchCandidate {
	var candidates []dispatchCandidate

	for _, order := range orders.Orders {
		if order.Status != OrderStatusActive && order.Status != OrderStatusFailing {
			continue
		}

		// Skip orders in busy/adopted/ticketed sets.
		if _, ok := busy[order.ID]; ok {
			continue
		}
		if _, ok := adopted[order.ID]; ok {
			continue
		}
		if _, ok := ticketed[order.ID]; ok {
			continue
		}

		// Skip failed orders — but "failing" orders are exempt (OnFailure must dispatch).
		if _, ok := failed[order.ID]; ok && order.Status != OrderStatusFailing {
			continue
		}

		stages := order.Stages
		isOnFailure := false
		if order.Status == OrderStatusFailing {
			stages = order.OnFailure
			isOnFailure = true
		}

		// Skip degenerate orders with empty stages.
		if len(stages) == 0 {
			continue
		}

		// Find first pending stage; skip if current stage is active (already dispatched).
		for i, s := range stages {
			if s.Status == StageStatusActive {
				// Already dispatched — order is busy at stage level.
				break
			}
			if s.Status == StageStatusPending {
				candidates = append(candidates, dispatchCandidate{
					OrderID:     order.ID,
					StageIndex:  i,
					Stage:       s,
					IsOnFailure: isOnFailure,
				})
				break
			}
		}
	}

	return candidates
}

// cloneOrdersFile returns a shallow copy of the OrdersFile with a new Orders slice.
func cloneOrdersFile(of OrdersFile) OrdersFile {
	newOrders := make([]Order, len(of.Orders))
	for i, o := range of.Orders {
		newOrders[i] = cloneOrder(o)
	}
	of.Orders = newOrders
	return of
}

// cloneOrder returns a copy of an Order with new stage slices.
func cloneOrder(o Order) Order {
	o.Stages = slices.Clone(o.Stages)
	if o.OnFailure != nil {
		o.OnFailure = slices.Clone(o.OnFailure)
	}
	if o.Plan != nil {
		o.Plan = slices.Clone(o.Plan)
	}
	return o
}

// loadOrders reads orders.json from disk into in-memory state.
func (l *Loop) loadOrders() error {
	of, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	l.orders = of
	l.ordersDirty = false
	return nil
}

// flushOrders writes the in-memory orders to disk atomically.
func (l *Loop) flushOrders() error {
	if !l.ordersDirty {
		return nil
	}
	if err := writeOrdersAtomic(l.deps.OrdersFile, l.orders); err != nil {
		return err
	}
	l.ordersDirty = false
	return nil
}

// markOrdersDirty flags the in-memory orders as needing a flush.
func (l *Loop) markOrdersDirty() {
	l.ordersDirty = true
}

func readOrders(path string) (OrdersFile, error) {
	of, err := orderx.ReadOrders(path)
	if err != nil {
		return OrdersFile{}, err
	}
	return fromOrdersFileX(of), nil
}

func writeOrdersAtomic(path string, of OrdersFile) error {
	return orderx.WriteOrdersAtomic(path, toOrdersFileX(of))
}

// consumeOrdersNext atomically promotes orders-next.json into orders.json.
//
// Reads and validates orders-next.json, merges into existing orders.json via
// WriteOrdersAtomic, THEN deletes orders-next.json. If the loop crashes after
// writing orders.json but before deleting orders-next.json, the next cycle
// re-promotes idempotently — duplicate order IDs across the two files are
// skipped (not rejected).
func consumeOrdersNext(nextPath, ordersPath string) (bool, error) {
	nextData, err := os.ReadFile(nextPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read orders-next: %w", err)
	}

	incoming, err := orderx.ParseOrdersStrict(nextData)
	if err != nil {
		// Rename invalid proposal so it doesn't block future cycles.
		// Preserve the file for debugging rather than deleting it.
		_ = os.Rename(nextPath, nextPath+".bad")
		return false, fmt.Errorf("invalid orders-next.json (renamed to .bad): %w", err)
	}

	// Read existing orders.
	existing, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		return false, fmt.Errorf("read existing orders: %w", err)
	}

	// Build set of existing order IDs for dedup.
	existingIDs := make(map[string]struct{}, len(existing.Orders))
	for _, order := range existing.Orders {
		existingIDs[order.ID] = struct{}{}
	}

	// Merge incoming orders, skipping duplicates.
	for _, order := range incoming.Orders {
		if _, exists := existingIDs[order.ID]; exists {
			continue
		}
		existing.Orders = append(existing.Orders, order)
		existingIDs[order.ID] = struct{}{}
	}

	// Write merged orders atomically.
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		return false, fmt.Errorf("promote orders-next.json: %w", err)
	}

	// Only delete after successful write — crash safety.
	if err := os.Remove(nextPath); err != nil && !os.IsNotExist(err) {
		return true, fmt.Errorf("remove orders-next.json: %w", err)
	}

	return true, nil
}

func toOrdersFileX(of OrdersFile) orderx.OrdersFile {
	orders := make([]orderx.Order, 0, len(of.Orders))
	for _, o := range of.Orders {
		orders = append(orders, toOrderX(o))
	}
	return orderx.OrdersFile{
		GeneratedAt:  of.GeneratedAt,
		Orders:       orders,
		ActionNeeded: of.ActionNeeded,
	}
}

func fromOrdersFileX(of orderx.OrdersFile) OrdersFile {
	orders := make([]Order, 0, len(of.Orders))
	for _, o := range of.Orders {
		orders = append(orders, fromOrderX(o))
	}
	return OrdersFile{
		GeneratedAt:  of.GeneratedAt,
		Orders:       orders,
		ActionNeeded: of.ActionNeeded,
	}
}

func toOrderX(o Order) orderx.Order {
	return orderx.Order{
		ID:        o.ID,
		Title:     o.Title,
		Plan:      o.Plan,
		Rationale: o.Rationale,
		Stages:    toStagesX(o.Stages),
		Status:    o.Status,
		OnFailure: toStagesX(o.OnFailure),
	}
}

func fromOrderX(o orderx.Order) Order {
	return Order{
		ID:        o.ID,
		Title:     o.Title,
		Plan:      o.Plan,
		Rationale: o.Rationale,
		Stages:    fromStagesX(o.Stages),
		Status:    o.Status,
		OnFailure: fromStagesX(o.OnFailure),
	}
}

func toStagesX(stages []Stage) []orderx.Stage {
	if stages == nil {
		return nil
	}
	out := make([]orderx.Stage, 0, len(stages))
	for _, s := range stages {
		out = append(out, orderx.Stage{
			TaskKey:  s.TaskKey,
			Prompt:   s.Prompt,
			Skill:    s.Skill,
			Provider: s.Provider,
			Model:    s.Model,
			Runtime:  s.Runtime,
			Status:   s.Status,
			Extra:    s.Extra,
		})
	}
	return out
}

func fromStagesX(stages []orderx.Stage) []Stage {
	if stages == nil {
		return nil
	}
	out := make([]Stage, 0, len(stages))
	for _, s := range stages {
		out = append(out, Stage{
			TaskKey:  s.TaskKey,
			Prompt:   s.Prompt,
			Skill:    s.Skill,
			Provider: s.Provider,
			Model:    s.Model,
			Runtime:  s.Runtime,
			Status:   s.Status,
			Extra:    s.Extra,
		})
	}
	return out
}

// NormalizeAndValidateOrders wraps the orderx function for loop-layer types.
func NormalizeAndValidateOrders(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool, error) {
	updated, changed, err := orderx.NormalizeAndValidateOrders(toOrdersFileX(of), reg, cfg)
	if err != nil {
		return OrdersFile{}, false, err
	}
	if !changed {
		return of, false, nil
	}
	return fromOrdersFileX(updated), true, nil
}

// ApplyOrderRoutingDefaults wraps the orderx function for loop-layer types.
func ApplyOrderRoutingDefaults(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool) {
	updated, changed := orderx.ApplyOrderRoutingDefaults(toOrdersFileX(of), reg, cfg)
	if !changed {
		return of, false
	}
	return fromOrdersFileX(updated), true
}
