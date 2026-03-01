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
	OrderID    string
	StageIndex int
	Stage      Stage
}

// activeOrderIDs returns active order IDs that still have pending work.
func activeOrderIDs(orders OrdersFile) []string {
	ids := make([]string, 0, len(orders.Orders))
	for _, order := range orders.Orders {
		if order.Status != OrderStatusActive {
			continue
		}
		for _, stage := range order.Stages {
			if stage.Status == StageStatusActive || stage.Status == StageStatusMerging || stage.Status == StageStatusPending {
				ids = append(ids, order.ID)
				break
			}
		}
	}
	return ids
}

// busyTargets returns order IDs currently blocked by an active stage.
func busyTargets(orders OrdersFile) map[string]bool {
	busy := make(map[string]bool)
	for _, order := range orders.Orders {
		if order.Status != OrderStatusActive {
			continue
		}
		for _, stage := range order.Stages {
			if stage.Status == StageStatusActive || stage.Status == StageStatusMerging {
				busy[order.ID] = true
				break
			}
			if stage.Status == StageStatusPending {
				break
			}
		}
	}
	return busy
}

// activeStageForOrder returns the index and pointer to the currently active or
// first pending stage. Returns (-1, nil) if no stage is active/pending.
func activeStageForOrder(order Order) (int, *Stage) {
	for i := range order.Stages {
		switch order.Stages[i].Status {
		case StageStatusActive, StageStatusMerging, StageStatusPending:
			return i, &order.Stages[i]
		}
	}
	return -1, nil
}

// advanceOrder marks the current active/first-pending stage as completed.
// If all stages complete, removes the order and returns removed=true.
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

	// Find and complete the current active/first-pending stage.
	advanced := false
	for i := range order.Stages {
		switch order.Stages[i].Status {
		case StageStatusActive, StageStatusMerging, StageStatusPending:
			order.Stages[i].Status = StageStatusCompleted
			advanced = true
		}
		if advanced {
			break
		}
	}
	if !advanced {
		return orders, false, fmt.Errorf("order %q has no active or pending stage to advance", orderID)
	}

	// Check if all stages are completed.
	allDone := true
	for _, s := range order.Stages {
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

// failStage marks the current active/pending stage as failed and marks the
// order as failed for explicit recovery via control commands.
func failStage(orders OrdersFile, orderID string, reason string) (OrdersFile, error) {
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

	// Mark current active/pending stage as failed.
	for i := range order.Stages {
		s := &order.Stages[i]
		if s.Status == StageStatusActive || s.Status == StageStatusMerging || s.Status == StageStatusPending {
			s.Status = StageStatusFailed
			break
		}
	}
	order.Status = OrderStatusFailed
	return orders, nil
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

	orders.Orders = slices.Delete(orders.Orders, idx, idx+1)
	return orders, nil
}

// dispatchableStages finds the first pending stage per order that is ready for dispatch.
// Orders in busy/adopted/ticketed sets are skipped.
func dispatchableStages(orders OrdersFile, busy, adopted, ticketed map[string]struct{}) []dispatchCandidate {
	var candidates []dispatchCandidate

	for _, order := range orders.Orders {
		if order.Status != OrderStatusActive {
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

		// Skip degenerate orders with empty stages.
		if len(order.Stages) == 0 {
			continue
		}

		// Find first pending stage; skip if current stage is active (already dispatched).
		for i, s := range order.Stages {
			if s.Status == StageStatusActive || s.Status == StageStatusMerging {
				// Already dispatched — order is busy at stage level.
				break
			}
			if s.Status == StageStatusPending {
				candidates = append(candidates, dispatchCandidate{
					OrderID:    order.ID,
					StageIndex: i,
					Stage:      s,
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
	if o.Plan != nil {
		o.Plan = slices.Clone(o.Plan)
	}
	return o
}

func readOrders(path string) (OrdersFile, error) {
	return orderx.ReadOrders(path)
}

func writeOrdersAtomic(path string, of OrdersFile) error {
	return orderx.WriteOrdersAtomic(path, of)
}

// consumeOrdersNext atomically promotes orders-next.json into orders.json.
//
// Returns (promoted, emptyPromotion, error). emptyPromotion is true when the
// incoming orders array was empty — the schedule agent ran but found nothing
// actionable.
//
// Reads and validates orders-next.json, merges into existing orders.json via
// WriteOrdersAtomic, THEN deletes orders-next.json. If the loop crashes after
// writing orders.json but before deleting orders-next.json, the next cycle
// re-promotes idempotently — duplicate order IDs across the two files are
// skipped (not rejected).
func consumeOrdersNext(nextPath, ordersPath string) (bool, bool, error) {
	nextData, err := os.ReadFile(nextPath)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read orders-next: %w", err)
	}

	incoming, err := orderx.ParseOrdersStrict(nextData)
	if err != nil {
		// Rename invalid proposal so it doesn't block future cycles.
		// Preserve the file for debugging rather than deleting it.
		_ = os.Rename(nextPath, nextPath+".bad")
		return false, false, fmt.Errorf("invalid orders-next.json (renamed to .bad): %w", err)
	}

	emptyPromotion := len(incoming.Orders) == 0

	// Read existing orders.
	existing, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		return false, false, fmt.Errorf("read existing orders: %w", err)
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
		return false, false, fmt.Errorf("promote orders-next.json: %w", err)
	}

	// Only delete after successful write — crash safety.
	if err := os.Remove(nextPath); err != nil && !os.IsNotExist(err) {
		return true, emptyPromotion, fmt.Errorf("remove orders-next.json: %w", err)
	}

	return true, emptyPromotion, nil
}

// NormalizeAndValidateOrders delegates to orderx.
func NormalizeAndValidateOrders(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool, error) {
	return orderx.NormalizeAndValidateOrders(of, reg, cfg)
}

// ApplyOrderRoutingDefaults delegates to orderx.
func ApplyOrderRoutingDefaults(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool) {
	return orderx.ApplyOrderRoutingDefaults(of, reg, cfg)
}
