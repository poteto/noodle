package loop

import (
	"fmt"
	"os"

	"github.com/poteto/noodle/internal/taskreg"
)

// RegistryDiff captures what changed between two registry snapshots.
type RegistryDiff struct {
	Added   []string
	Removed []string
}

// diffRegistryKeys compares old and new registry key sets.
func diffRegistryKeys(old, new taskreg.Registry) RegistryDiff {
	oldKeys := make(map[string]struct{})
	for _, tt := range old.All() {
		oldKeys[tt.Key] = struct{}{}
	}
	newKeys := make(map[string]struct{})
	for _, tt := range new.All() {
		newKeys[tt.Key] = struct{}{}
	}

	var added []string
	for k := range newKeys {
		if _, ok := oldKeys[k]; !ok {
			added = append(added, k)
		}
	}
	var removed []string
	for k := range oldKeys {
		if _, ok := newKeys[k]; !ok {
			removed = append(removed, k)
		}
	}
	return RegistryDiff{Added: added, Removed: removed}
}

// auditOrders checks each order's stages against the current registry.
// Orders with no resolvable stages are dropped. Emits order.dropped events.
func (l *Loop) auditOrders() {
	orders, err := l.currentOrders()
	if err != nil {
		return
	}

	var kept []Order
	var droppedIDs []string
	for _, order := range orders.Orders {
		hasValid := false
		for _, stage := range order.Stages {
			input := taskreg.StageInput{
				TaskKey: stage.TaskKey,
				Skill:   stage.Skill,
			}
			if _, ok := l.registry.ResolveStage(input); ok {
				hasValid = true
				break
			}
		}
		if hasValid {
			kept = append(kept, order)
		} else {
			droppedIDs = append(droppedIDs, order.ID)
			fmt.Fprintf(os.Stderr, "dropped order %q: no stages resolve\n", order.ID)
		}
	}

	if len(droppedIDs) == 0 {
		return
	}

	orders.Orders = kept
	if err := l.writeOrdersState(orders); err != nil {
		fmt.Fprintf(os.Stderr, "order-audit: write orders: %v\n", err)
		return
	}

	for _, id := range droppedIDs {
		_ = l.events.Emit(LoopEventOrderDropped, OrderDroppedPayload{
			OrderID: id,
			Reason:  "no stages resolve",
		})
	}
}
