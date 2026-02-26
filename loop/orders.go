package loop

import (
	"fmt"
	"os"

	"github.com/poteto/noodle/internal/queuex"
)

func readOrders(path string) (OrdersFile, error) {
	of, err := queuex.ReadOrders(path)
	if err != nil {
		return OrdersFile{}, err
	}
	return fromOrdersFileX(of), nil
}

func writeOrdersAtomic(path string, of OrdersFile) error {
	return queuex.WriteOrdersAtomic(path, toOrdersFileX(of))
}

// consumeOrdersNext atomically promotes orders-next.json into orders.json.
//
// Unlike consumeQueueNext (which deletes next, then writes), this function
// reads and validates orders-next.json, merges into existing orders.json via
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

	incoming, err := queuex.ParseOrdersStrict(nextData)
	if err != nil {
		// Remove invalid proposal so it doesn't block future cycles.
		_ = os.Remove(nextPath)
		return false, fmt.Errorf("invalid orders-next.json (removed): %w", err)
	}

	// Read existing orders.
	existing, err := queuex.ReadOrders(ordersPath)
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
	if err := queuex.WriteOrdersAtomic(ordersPath, existing); err != nil {
		return false, fmt.Errorf("promote orders-next.json: %w", err)
	}

	// Only delete after successful write — crash safety.
	if err := os.Remove(nextPath); err != nil && !os.IsNotExist(err) {
		return true, fmt.Errorf("remove orders-next.json: %w", err)
	}

	return true, nil
}

func toOrdersFileX(of OrdersFile) queuex.OrdersFile {
	orders := make([]queuex.Order, 0, len(of.Orders))
	for _, o := range of.Orders {
		orders = append(orders, toOrderX(o))
	}
	return queuex.OrdersFile{
		GeneratedAt:  of.GeneratedAt,
		Orders:       orders,
		ActionNeeded: of.ActionNeeded,
	}
}

func fromOrdersFileX(of queuex.OrdersFile) OrdersFile {
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

func toOrderX(o Order) queuex.Order {
	return queuex.Order{
		ID:        o.ID,
		Title:     o.Title,
		Plan:      o.Plan,
		Rationale: o.Rationale,
		Stages:    toStagesX(o.Stages),
		Status:    o.Status,
		OnFailure: toStagesX(o.OnFailure),
	}
}

func fromOrderX(o queuex.Order) Order {
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

func toStagesX(stages []Stage) []queuex.Stage {
	if stages == nil {
		return nil
	}
	out := make([]queuex.Stage, 0, len(stages))
	for _, s := range stages {
		out = append(out, queuex.Stage{
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

func fromStagesX(stages []queuex.Stage) []Stage {
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
