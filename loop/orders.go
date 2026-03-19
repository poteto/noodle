package loop

import (
	"errors"
	"fmt"
	"os"
	"reflect"
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

type ordersNextRejectedError struct {
	cause error
}

func (e ordersNextRejectedError) Error() string {
	if e.cause == nil {
		return "orders-next rejected"
	}
	return e.cause.Error()
}

func (e ordersNextRejectedError) Unwrap() error {
	return e.cause
}

func isOrdersNextRejectedError(err error) bool {
	var target ordersNextRejectedError
	return errors.As(err, &target)
}

// consumeOrdersNext atomically promotes orders-next.json into orders.json.
//
// Returns the merged in-memory result; the caller persists it and removes
// orders-next.json after a successful write for crash safety.
type mergeResult struct {
	Orders         OrdersFile
	Promoted       bool
	EmptyPromotion bool
}

// Reads and validates orders-next.json, merges into the provided orders, and
// returns the merged result. If the loop crashes after writing orders.json but
// before deleting orders-next.json, the next cycle
// re-promotes idempotently — duplicate order IDs across the two files are
// skipped (not rejected), except when replacing a failed order with a new
// active proposal for explicit restart.
func consumeOrdersNext(nextPath string, existing OrdersFile) (mergeResult, error) {
	nextData, err := os.ReadFile(nextPath)
	if os.IsNotExist(err) {
		return mergeResult{}, nil
	}
	if err != nil {
		return mergeResult{}, fmt.Errorf("read orders-next: %w", err)
	}

	compact, err := orderx.ParseCompactOrders(nextData)
	if err != nil {
		// Rename invalid proposal so it doesn't block future cycles.
		// Preserve the file for debugging rather than deleting it.
		_ = os.Rename(nextPath, nextPath+".bad")
		cause := fmt.Errorf("invalid orders-next.json (renamed to .bad): %w", err)
		return mergeResult{}, ordersNextRejectedError{cause: cause}
	}
	incoming, err := orderx.ExpandCompactOrders(compact)
	if err != nil {
		_ = os.Rename(nextPath, nextPath+".bad")
		cause := fmt.Errorf("invalid orders-next.json (renamed to .bad): %w", err)
		return mergeResult{}, ordersNextRejectedError{cause: cause}
	}

	emptyPromotion := len(incoming.Orders) == 0

	// Build index of existing order IDs for dedup/replacement decisions.
	existingIndex := make(map[string]int, len(existing.Orders))
	for i, order := range existing.Orders {
		existingIndex[order.ID] = i
	}

	// Merge incoming orders. Duplicates are skipped for crash-safe idempotency,
	// except:
	// 1) a failed existing order can be replaced by a new active proposal
	// 2) an active existing order can be amended by a new active proposal
	for _, order := range incoming.Orders {
		idx, exists := existingIndex[order.ID]
		if exists {
			if shouldReplaceFailedOrder(existing.Orders[idx], order) {
				existing.Orders[idx] = order
				continue
			}
			merged, replaced := mergeAmendedActiveOrder(existing.Orders[idx], order)
			if replaced {
				existing.Orders[idx] = merged
			}
			continue
		}
		existing.Orders = append(existing.Orders, order)
		existingIndex[order.ID] = len(existing.Orders) - 1
	}

	return mergeResult{
		Orders:         existing,
		Promoted:       true,
		EmptyPromotion: emptyPromotion,
	}, nil
}

func shouldReplaceFailedOrder(existing Order, incoming Order) bool {
	return existing.Status == OrderStatusFailed && incoming.Status == OrderStatusActive
}

func shouldAmendActiveOrder(existing Order, incoming Order) bool {
	return existing.Status == OrderStatusActive && incoming.Status == OrderStatusActive
}

func mergeAmendedActiveOrder(existing Order, incoming Order) (Order, bool) {
	if !shouldAmendActiveOrder(existing, incoming) {
		return existing, false
	}
	currentIndex, currentStage := activeStageForOrder(existing)
	if currentIndex < 0 || currentStage == nil {
		return existing, false
	}

	merged := incoming
	completedPrefix := slices.Clone(existing.Stages[:currentIndex])
	matchIndex := firstMatchingStageIndex(incoming.Stages, *currentStage)
	if matchIndex >= 0 {
		merged.Stages = append(completedPrefix, *currentStage)
		merged.Stages = append(merged.Stages, incoming.Stages[matchIndex+1:]...)
	} else {
		sharedPrefix := matchingPrefixLen(existing.Stages[:currentIndex], incoming.Stages)
		merged.Stages = append(completedPrefix, incoming.Stages[sharedPrefix:]...)
	}
	if len(merged.Stages) == 0 {
		return existing, false
	}
	merged.Status = existing.Status
	if reflect.DeepEqual(existing, merged) {
		return existing, false
	}
	return merged, true
}

func firstMatchingStageIndex(stages []Stage, target Stage) int {
	for i := range stages {
		if sameStageDefinition(stages[i], target) {
			return i
		}
	}
	return -1
}

func matchingPrefixLen(existingPrefix []Stage, incoming []Stage) int {
	limit := len(existingPrefix)
	if len(incoming) < limit {
		limit = len(incoming)
	}
	for i := 0; i < limit; i++ {
		if !sameStageDefinition(existingPrefix[i], incoming[i]) {
			return i
		}
	}
	return limit
}

func sameStageDefinition(a Stage, b Stage) bool {
	a.Status = ""
	b.Status = ""
	return reflect.DeepEqual(a, b)
}

// NormalizeAndValidateOrders delegates to orderx.
func NormalizeAndValidateOrders(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool, error) {
	return orderx.NormalizeAndValidateOrders(of, reg, cfg)
}

// ApplyOrderRoutingDefaults delegates to orderx.
func ApplyOrderRoutingDefaults(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool) {
	return orderx.ApplyOrderRoutingDefaults(of, reg, cfg)
}
