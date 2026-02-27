package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/orderx"
)

func TestReadWriteOrdersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orders.json")

	now := time.Now().Truncate(time.Second)
	original := OrdersFile{
		GeneratedAt: now,
		Orders: []Order{
			{
				ID:     "1",
				Title:  "implement feature",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
						Extra: map[string]json.RawMessage{
							"priority": json.RawMessage(`42`),
						},
					},
				},
				OnFailure: []Stage{
					{
						TaskKey:  "execute",
						Prompt:   "rollback",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
		},
		ActionNeeded: []string{"check order 1"},
	}

	if err := writeOrdersAtomic(path, original); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readOrders(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !got.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt mismatch")
	}
	if len(got.Orders) != 1 {
		t.Fatalf("Orders len = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].ID != "1" {
		t.Errorf("ID = %q, want 1", got.Orders[0].ID)
	}
	if string(got.Orders[0].Stages[0].Extra["priority"]) != `42` {
		t.Errorf("Extra[priority] lost in round-trip")
	}
	if len(got.Orders[0].OnFailure) != 1 {
		t.Fatalf("OnFailure len = %d, want 1", len(got.Orders[0].OnFailure))
	}
}

func TestConsumeOrdersNextPromotesFile(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	// Write existing orders.
	existing := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "existing-1", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	// Write next.
	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "new-1", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	// orders.json should contain both.
	got, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Orders) != 2 {
		t.Fatalf("expected 2 orders after merge, got %d", len(got.Orders))
	}
	ids := map[string]bool{}
	for _, o := range got.Orders {
		ids[o.ID] = true
	}
	if !ids["existing-1"] || !ids["new-1"] {
		t.Fatalf("missing expected IDs, got: %v", ids)
	}

	// orders-next.json should be gone.
	if _, err := os.Stat(nextPath); !os.IsNotExist(err) {
		t.Fatal("orders-next.json should be removed")
	}
}

func TestConsumeOrdersNextMissingReturnsNoop(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if promoted {
		t.Fatal("expected promoted=false when orders-next missing")
	}
}

func TestConsumeOrdersNextDuplicateIDsSkipped(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	// Write existing with ID "dup".
	existing := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "dup", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
			{ID: "keep", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	// Write next with duplicate ID "dup" and new ID "new".
	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "dup", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
			{ID: "new", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	got, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	// Should have 3 orders: dup (from existing), keep, new.
	if len(got.Orders) != 3 {
		t.Fatalf("expected 3 orders (dup skipped), got %d", len(got.Orders))
	}

	// Verify no duplicate IDs.
	seen := map[string]int{}
	for _, o := range got.Orders {
		seen[o.ID]++
	}
	if seen["dup"] != 1 {
		t.Errorf("expected dup to appear once, got %d", seen["dup"])
	}
}

func TestConsumeOrdersNextCrashSafety(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	// Simulate: orders.json already has merged content, but orders-next.json
	// still exists (crash between write and delete).
	merged := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "original", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
			{ID: "incoming", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, merged); err != nil {
		t.Fatal(err)
	}

	// orders-next.json still has the incoming order (crash replay).
	leftover := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "incoming", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, leftover); err != nil {
		t.Fatal(err)
	}

	// Re-run: should be idempotent.
	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true (file existed)")
	}

	got, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	// No duplicates — still 2 orders.
	if len(got.Orders) != 2 {
		t.Fatalf("expected 2 orders after idempotent re-promote, got %d", len(got.Orders))
	}
	seen := map[string]int{}
	for _, o := range got.Orders {
		seen[o.ID]++
	}
	if seen["incoming"] != 1 {
		t.Errorf("expected incoming to appear once, got %d", seen["incoming"])
	}

	// orders-next.json should be cleaned up.
	if _, err := os.Stat(nextPath); !os.IsNotExist(err) {
		t.Fatal("orders-next.json should be removed after re-promote")
	}
}

func TestConsumeOrdersNextInvalidNextJSON(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	if err := os.WriteFile(nextPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if promoted {
		t.Fatal("expected promoted=false on error")
	}
	if !strings.Contains(err.Error(), "invalid orders-next.json") {
		t.Fatalf("expected descriptive error, got: %v", err)
	}

	// Invalid file should be removed.
	if _, err := os.Stat(nextPath); !os.IsNotExist(err) {
		t.Fatal("invalid orders-next.json should be removed")
	}
}

func TestConsumeOrdersNextNoExistingOrders(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	// No existing orders.json.
	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{ID: "first", Status: orderx.OrderStatusActive, Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}}},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	got, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Orders) != 1 || got.Orders[0].ID != "first" {
		t.Fatalf("expected 1 order with ID first, got: %+v", got.Orders)
	}
}

// --- Stage lifecycle function tests ---

func makeStage(status string) Stage {
	return Stage{
		TaskKey:  "execute",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Status:   status,
	}
}

func makeOrder(id, status string, stages []Stage, onFailure []Stage) Order {
	return Order{
		ID:        id,
		Title:     "order " + id,
		Status:    status,
		Stages:    stages,
		OnFailure: onFailure,
	}
}

func TestActiveStageForOrder(t *testing.T) {
	t.Run("returns first pending", func(t *testing.T) {
		order := makeOrder("1", OrderStatusActive, []Stage{
			makeStage(StageStatusCompleted),
			makeStage(StageStatusPending),
			makeStage(StageStatusPending),
		}, nil)
		idx, stage := activeStageForOrder(order)
		if idx != 1 {
			t.Errorf("idx = %d, want 1", idx)
		}
		if stage == nil {
			t.Fatal("stage is nil")
		}
		if stage.Status != StageStatusPending {
			t.Errorf("status = %q, want pending", stage.Status)
		}
	})

	t.Run("returns active stage", func(t *testing.T) {
		order := makeOrder("1", OrderStatusActive, []Stage{
			makeStage(StageStatusCompleted),
			makeStage(StageStatusActive),
			makeStage(StageStatusPending),
		}, nil)
		idx, stage := activeStageForOrder(order)
		if idx != 1 {
			t.Errorf("idx = %d, want 1", idx)
		}
		if stage.Status != StageStatusActive {
			t.Errorf("status = %q, want active", stage.Status)
		}
	})

	t.Run("all completed returns nil", func(t *testing.T) {
		order := makeOrder("1", OrderStatusActive, []Stage{
			makeStage(StageStatusCompleted),
			makeStage(StageStatusCompleted),
		}, nil)
		idx, stage := activeStageForOrder(order)
		if idx != -1 || stage != nil {
			t.Errorf("expected (-1, nil), got (%d, %v)", idx, stage)
		}
	})

	t.Run("failing order checks OnFailure", func(t *testing.T) {
		order := makeOrder("1", OrderStatusFailing, []Stage{
			makeStage(StageStatusFailed),
		}, []Stage{
			makeStage(StageStatusCompleted),
			makeStage(StageStatusPending),
		})
		idx, stage := activeStageForOrder(order)
		if idx != 1 {
			t.Errorf("idx = %d, want 1", idx)
		}
		if stage.Status != StageStatusPending {
			t.Errorf("status = %q, want pending", stage.Status)
		}
	})
}

func TestAdvanceOrder(t *testing.T) {
	t.Run("three stages advance through each", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{
					makeStage(StageStatusPending),
					makeStage(StageStatusPending),
					makeStage(StageStatusPending),
				}, nil),
			},
		}

		// Advance stage 0.
		var removed bool
		var err error
		of, removed, err = advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Fatal("order removed too early")
		}
		if of.Orders[0].Stages[0].Status != StageStatusCompleted {
			t.Errorf("stage 0 = %q, want completed", of.Orders[0].Stages[0].Status)
		}

		// Advance stage 1.
		of, removed, err = advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Fatal("order removed too early")
		}
		if of.Orders[0].Stages[1].Status != StageStatusCompleted {
			t.Errorf("stage 1 = %q, want completed", of.Orders[0].Stages[1].Status)
		}

		// Advance stage 2 — should remove order.
		of, removed, err = advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Fatal("expected removed=true on final stage")
		}
		if len(of.Orders) != 0 {
			t.Errorf("expected 0 orders, got %d", len(of.Orders))
		}
	})

	t.Run("final stage removes order", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("keep", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("remove", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
			},
		}

		of, removed, err := advanceOrder(of, "remove")
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Fatal("expected removed=true")
		}
		if len(of.Orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(of.Orders))
		}
		if of.Orders[0].ID != "keep" {
			t.Errorf("remaining order = %q, want keep", of.Orders[0].ID)
		}
	})

	t.Run("missing order returns error", func(t *testing.T) {
		of := OrdersFile{Orders: []Order{
			makeOrder("1", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
		}}

		_, _, err := advanceOrder(of, "nonexistent")
		if err == nil {
			t.Fatal("expected error for missing order")
		}
	})

	t.Run("advances active stage", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}, nil),
			},
		}
		of, removed, err := advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Fatal("unexpected removal")
		}
		if of.Orders[0].Stages[1].Status != StageStatusCompleted {
			t.Errorf("stage 1 = %q, want completed", of.Orders[0].Stages[1].Status)
		}
	})

	t.Run("failing order advances OnFailure stages", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusFailing, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusFailed),
					makeStage(StageStatusCancelled),
				}, []Stage{
					makeStage(StageStatusPending),
					makeStage(StageStatusPending),
				}),
			},
		}

		// Advance first OnFailure stage.
		of, removed, err := advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if removed {
			t.Fatal("removed too early")
		}
		if of.Orders[0].OnFailure[0].Status != StageStatusCompleted {
			t.Errorf("OnFailure[0] = %q, want completed", of.Orders[0].OnFailure[0].Status)
		}

		// Advance second (last) OnFailure stage — removes order.
		of, removed, err = advanceOrder(of, "1")
		if err != nil {
			t.Fatal(err)
		}
		if !removed {
			t.Fatal("expected removed=true on last OnFailure stage")
		}
		if len(of.Orders) != 0 {
			t.Errorf("expected 0 orders, got %d", len(of.Orders))
		}
	})
}

func TestFailStage(t *testing.T) {
	t.Run("no OnFailure removes order", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}, nil),
			},
		}

		of, terminal, err := failStage(of, "1", "test failure")
		if err != nil {
			t.Fatal(err)
		}
		if !terminal {
			t.Fatal("expected terminal=true with no OnFailure")
		}
		if len(of.Orders) != 0 {
			t.Fatalf("expected 0 orders, got %d", len(of.Orders))
		}
	})

	t.Run("with OnFailure sets failing", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}, []Stage{
					makeStage(StageStatusPending),
					makeStage(StageStatusPending),
				}),
			},
		}

		of, terminal, err := failStage(of, "1", "test failure")
		if err != nil {
			t.Fatal(err)
		}
		if terminal {
			t.Fatal("expected terminal=false when OnFailure exists")
		}
		if len(of.Orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(of.Orders))
		}

		order := of.Orders[0]
		if order.Status != OrderStatusFailing {
			t.Errorf("status = %q, want failing", order.Status)
		}
		// Main stages: completed, failed, cancelled.
		if order.Stages[0].Status != StageStatusCompleted {
			t.Errorf("stage 0 = %q, want completed", order.Stages[0].Status)
		}
		if order.Stages[1].Status != StageStatusFailed {
			t.Errorf("stage 1 = %q, want failed", order.Stages[1].Status)
		}
		if order.Stages[2].Status != StageStatusCancelled {
			t.Errorf("stage 2 = %q, want cancelled", order.Stages[2].Status)
		}
		// OnFailure stages should be pending.
		for i, s := range order.OnFailure {
			if s.Status != StageStatusPending {
				t.Errorf("OnFailure[%d] = %q, want pending", i, s.Status)
			}
		}
	})

	t.Run("already failing removes order", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusFailing, []Stage{
					makeStage(StageStatusFailed),
					makeStage(StageStatusCancelled),
				}, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
				}),
			},
		}

		of, terminal, err := failStage(of, "1", "OnFailure also failed")
		if err != nil {
			t.Fatal(err)
		}
		if !terminal {
			t.Fatal("expected terminal=true when already failing")
		}
		if len(of.Orders) != 0 {
			t.Fatalf("expected 0 orders, got %d", len(of.Orders))
		}
	})

	t.Run("missing order returns error", func(t *testing.T) {
		of := OrdersFile{}
		_, _, err := failStage(of, "nonexistent", "reason")
		if err == nil {
			t.Fatal("expected error for missing order")
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Run("mix of completed and pending", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("keep", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("cancel", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}, []Stage{
					makeStage(StageStatusPending),
				}),
			},
		}

		of, err := cancelOrder(of, "cancel")
		if err != nil {
			t.Fatal(err)
		}

		// Order should be removed.
		if len(of.Orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(of.Orders))
		}
		if of.Orders[0].ID != "keep" {
			t.Errorf("remaining order = %q, want keep", of.Orders[0].ID)
		}
	})

	t.Run("missing order returns error", func(t *testing.T) {
		of := OrdersFile{}
		_, err := cancelOrder(of, "nonexistent")
		if err == nil {
			t.Fatal("expected error for missing order")
		}
	})
}

func TestDispatchableStages(t *testing.T) {
	empty := map[string]struct{}{}

	t.Run("returns first pending stage per order", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("a", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusPending),
					makeStage(StageStatusPending),
				}, nil),
				makeOrder("b", OrderStatusActive, []Stage{
					makeStage(StageStatusPending),
				}, nil),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 2 {
			t.Fatalf("expected 2 candidates, got %d", len(candidates))
		}
		if candidates[0].OrderID != "a" || candidates[0].StageIndex != 1 {
			t.Errorf("candidate 0 = %+v", candidates[0])
		}
		if candidates[1].OrderID != "b" || candidates[1].StageIndex != 0 {
			t.Errorf("candidate 1 = %+v", candidates[1])
		}
		if candidates[0].IsOnFailure || candidates[1].IsOnFailure {
			t.Error("IsOnFailure should be false for active orders")
		}
	})

	t.Run("skips busy/failed/adopted/ticketed", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("busy", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("failed", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("adopted", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("ticketed", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("free", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
			},
		}

		busySet := map[string]struct{}{"busy": {}}
		failedSet := map[string]struct{}{"failed": {}}
		adoptedSet := map[string]struct{}{"adopted": {}}
		ticketedSet := map[string]struct{}{"ticketed": {}}

		candidates := dispatchableStages(of, busySet, failedSet, adoptedSet, ticketedSet)
		if len(candidates) != 1 {
			t.Fatalf("expected 1 candidate, got %d", len(candidates))
		}
		if candidates[0].OrderID != "free" {
			t.Errorf("candidate = %q, want free", candidates[0].OrderID)
		}
	})

	t.Run("skips orders with active stage", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("dispatched", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}, nil),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 0 {
			t.Fatalf("expected 0 candidates (active stage = already dispatched), got %d", len(candidates))
		}
	})

	t.Run("dispatches OnFailure for failing orders", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusFailing, []Stage{
					makeStage(StageStatusFailed),
					makeStage(StageStatusCancelled),
				}, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusPending),
				}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 1 {
			t.Fatalf("expected 1 candidate, got %d", len(candidates))
		}
		if !candidates[0].IsOnFailure {
			t.Error("expected IsOnFailure=true")
		}
		if candidates[0].StageIndex != 1 {
			t.Errorf("StageIndex = %d, want 1", candidates[0].StageIndex)
		}
	})

	t.Run("exempts failing from failed set", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusFailing, []Stage{
					makeStage(StageStatusFailed),
				}, []Stage{
					makeStage(StageStatusPending),
				}),
			},
		}

		failedSet := map[string]struct{}{"1": {}}
		candidates := dispatchableStages(of, empty, failedSet, empty, empty)
		if len(candidates) != 1 {
			t.Fatalf("expected 1 candidate (failing exempt from failed), got %d", len(candidates))
		}
		if candidates[0].OrderID != "1" {
			t.Errorf("candidate = %q, want 1", candidates[0].OrderID)
		}
	})

	t.Run("skips degenerate empty stages", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("empty", OrderStatusActive, []Stage{}, nil),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 0 {
			t.Fatalf("expected 0 candidates for degenerate order, got %d", len(candidates))
		}
	})

	t.Run("skips failing order with empty OnFailure", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusFailing, []Stage{
					makeStage(StageStatusFailed),
				}, []Stage{}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 0 {
			t.Fatalf("expected 0 candidates for failing order with empty OnFailure, got %d", len(candidates))
		}
	})

	t.Run("respects array ordering", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("first", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("second", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
				makeOrder("third", OrderStatusActive, []Stage{makeStage(StageStatusPending)}, nil),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty, empty)
		if len(candidates) != 3 {
			t.Fatalf("expected 3 candidates, got %d", len(candidates))
		}
		if candidates[0].OrderID != "first" || candidates[1].OrderID != "second" || candidates[2].OrderID != "third" {
			t.Errorf("order mismatch: %q, %q, %q", candidates[0].OrderID, candidates[1].OrderID, candidates[2].OrderID)
		}
	})
}

func TestActiveOrderIDs(t *testing.T) {
	of := OrdersFile{
		Orders: []Order{
			makeOrder("active-pending", OrderStatusActive, []Stage{
				makeStage(StageStatusPending),
			}, nil),
			makeOrder("active-active", OrderStatusActive, []Stage{
				makeStage(StageStatusActive),
			}, nil),
			makeOrder("failing-pending", OrderStatusFailing, []Stage{
				makeStage(StageStatusFailed),
			}, []Stage{
				makeStage(StageStatusPending),
			}),
			makeOrder("done", OrderStatusCompleted, []Stage{
				makeStage(StageStatusCompleted),
			}, nil),
			makeOrder("failing-empty", OrderStatusFailing, []Stage{
				makeStage(StageStatusFailed),
			}, []Stage{}),
		},
	}

	got := ActiveOrderIDs(of)
	want := []string{"active-pending", "active-active", "failing-pending"}
	if len(got) != len(want) {
		t.Fatalf("ActiveOrderIDs len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ActiveOrderIDs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBusyTargets(t *testing.T) {
	of := OrdersFile{
		Orders: []Order{
			makeOrder("busy-main", OrderStatusActive, []Stage{
				makeStage(StageStatusCompleted),
				makeStage(StageStatusActive),
				makeStage(StageStatusPending),
			}, nil),
			makeOrder("not-busy-main", OrderStatusActive, []Stage{
				makeStage(StageStatusCompleted),
				makeStage(StageStatusPending),
			}, nil),
			makeOrder("busy-failing", OrderStatusFailing, []Stage{
				makeStage(StageStatusFailed),
			}, []Stage{
				makeStage(StageStatusActive),
			}),
			makeOrder("not-busy-failing", OrderStatusFailing, []Stage{
				makeStage(StageStatusFailed),
			}, []Stage{
				makeStage(StageStatusPending),
			}),
		},
	}

	busy := BusyTargets(of)
	if !busy["busy-main"] {
		t.Fatal("expected busy-main to be busy")
	}
	if !busy["busy-failing"] {
		t.Fatal("expected busy-failing to be busy")
	}
	if busy["not-busy-main"] {
		t.Fatal("did not expect not-busy-main to be busy")
	}
	if busy["not-busy-failing"] {
		t.Fatal("did not expect not-busy-failing to be busy")
	}
}

// Test value semantics — mutations should not affect original.
func TestLifecycleValueSemantics(t *testing.T) {
	original := OrdersFile{
		Orders: []Order{
			makeOrder("1", OrderStatusActive, []Stage{
				makeStage(StageStatusPending),
				makeStage(StageStatusPending),
			}, nil),
		},
	}

	advanced, _, err := advanceOrder(original, "1")
	if err != nil {
		t.Fatal(err)
	}

	// Original should be unmodified.
	if original.Orders[0].Stages[0].Status != StageStatusPending {
		t.Error("advanceOrder mutated original OrdersFile")
	}
	if advanced.Orders[0].Stages[0].Status != StageStatusCompleted {
		t.Error("advanceOrder did not complete stage in returned value")
	}
}

func TestOrdersFileCloneRoundTrip(t *testing.T) {
	original := OrdersFile{
		GeneratedAt: time.Now().Truncate(time.Second),
		Orders: []Order{
			{
				ID:        "1",
				Title:     "test",
				Plan:      []string{"step1"},
				Rationale: "because",
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Prompt:   "do thing",
						Skill:    "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Runtime:  "tmux",
						Status:   StageStatusPending,
						Extra: map[string]json.RawMessage{
							"key": json.RawMessage(`"value"`),
						},
					},
				},
				Status: OrderStatusActive,
				OnFailure: []Stage{
					{
						TaskKey:  "execute",
						Prompt:   "rollback",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
		},
		ActionNeeded: []string{"check"},
	}

	roundTrip := cloneOrdersFile(original)
	if len(roundTrip.Orders) != 1 {
		t.Fatalf("Orders len = %d", len(roundTrip.Orders))
	}
	o := roundTrip.Orders[0]
	if o.ID != "1" || o.Title != "test" || o.Rationale != "because" {
		t.Errorf("basic fields mismatch")
	}
	if len(o.Plan) != 1 || o.Plan[0] != "step1" {
		t.Errorf("Plan mismatch")
	}
	if len(o.Stages) != 1 {
		t.Fatalf("Stages len = %d", len(o.Stages))
	}
	s := o.Stages[0]
	if s.TaskKey != "execute" || s.Provider != "claude" || s.Runtime != "tmux" {
		t.Errorf("stage fields mismatch")
	}
	if string(s.Extra["key"]) != `"value"` {
		t.Errorf("Extra lost in clone round-trip")
	}
	if len(o.OnFailure) != 1 || o.OnFailure[0].Prompt != "rollback" {
		t.Errorf("OnFailure mismatch")
	}
}
