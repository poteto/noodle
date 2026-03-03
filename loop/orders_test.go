package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
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

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

func TestConsumeOrdersNextDefaultsMissingLifecycleStatuses(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	// Simulate compact scheduler output where order/stage status fields are omitted.
	nextData := []byte(`{
  "orders": [
    {
      "id": "108",
      "title": "compact wire order",
      "stages": [
        {
          "task_key": "execute",
          "provider": "claude",
          "model": "claude-opus-4-6"
        }
      ]
    }
  ]
}`)
	if err := os.WriteFile(nextPath, nextData, 0o644); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	got, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatalf("read promoted orders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders len = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].Status != orderx.OrderStatusActive {
		t.Fatalf("order status = %q, want %q", got.Orders[0].Status, orderx.OrderStatusActive)
	}
	if len(got.Orders[0].Stages) != 1 {
		t.Fatalf("stages len = %d, want 1", len(got.Orders[0].Stages))
	}
	if got.Orders[0].Stages[0].Status != orderx.StageStatusPending {
		t.Fatalf("stage status = %q, want %q", got.Orders[0].Stages[0].Status, orderx.StageStatusPending)
	}
}

func TestConsumeOrdersNextMissingReturnsNoop(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

func TestConsumeOrdersNextDuplicateFailedIDReplaced(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	existing := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "old failed order",
				Status: orderx.OrderStatusFailed,
				Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusFailed}},
			},
			{
				ID:     "keep",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "restarted order",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending}},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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
	if len(got.Orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(got.Orders))
	}

	var replaced *orderx.Order
	for i := range got.Orders {
		if got.Orders[i].ID == "dup" {
			replaced = &got.Orders[i]
			break
		}
	}
	if replaced == nil {
		t.Fatal("expected dup order to exist after promotion")
	}
	if replaced.Status != orderx.OrderStatusActive {
		t.Fatalf("dup order status = %q, want %q", replaced.Status, orderx.OrderStatusActive)
	}
	if replaced.Title != "restarted order" {
		t.Fatalf("dup order title = %q, want %q", replaced.Title, "restarted order")
	}
	if len(replaced.Stages) != 1 || replaced.Stages[0].Status != orderx.StageStatusPending {
		t.Fatalf("dup order stages not replaced: %+v", replaced.Stages)
	}
}

func TestConsumeOrdersNextDuplicateActiveIDAmendsPendingTail(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	existing := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "existing order",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{
					{TaskKey: "execute", Prompt: "ship this", Provider: "codex", Model: "gpt-5.3-codex", Status: orderx.StageStatusActive},
					{TaskKey: "quality", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
					{TaskKey: "reflect", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
				},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "amended order",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{
					{TaskKey: "execute", Prompt: "ship this", Provider: "codex", Model: "gpt-5.3-codex", Status: orderx.StageStatusPending},
					{TaskKey: "adversarial-review", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
				},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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
	if len(got.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(got.Orders))
	}
	stages := got.Orders[0].Stages
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages after amendment, got %d", len(stages))
	}
	if stages[0].TaskKey != "execute" || stages[0].Status != orderx.StageStatusActive {
		t.Fatalf("stage[0] = %+v, want active execute", stages[0])
	}
	if stages[1].TaskKey != "adversarial-review" || stages[1].Status != orderx.StageStatusPending {
		t.Fatalf("stage[1] = %+v, want pending adversarial-review", stages[1])
	}
}

func TestConsumeOrdersNextDuplicateActiveIDAmendsCurrentStage(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")
	nextPath := filepath.Join(dir, "orders-next.json")

	existing := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "existing order",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{
					{TaskKey: "execute", Prompt: "old prompt", Provider: "codex", Model: "gpt-5.3-codex", Status: orderx.StageStatusActive},
					{TaskKey: "quality", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
				},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	next := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "dup",
				Title:  "amended order",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{
					{TaskKey: "execute", Prompt: "new prompt", Provider: "codex", Model: "gpt-5.3-codex", Status: orderx.StageStatusPending},
					{TaskKey: "adversarial-review", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
				},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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
	if len(got.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(got.Orders))
	}
	stages := got.Orders[0].Stages
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages after amendment, got %d", len(stages))
	}
	if stages[0].TaskKey != "execute" || stages[0].Prompt != "new prompt" || stages[0].Status != orderx.StageStatusPending {
		t.Fatalf("stage[0] = %+v, want pending amended execute stage", stages[0])
	}
	if stages[1].TaskKey != "adversarial-review" || stages[1].Status != orderx.StageStatusPending {
		t.Fatalf("stage[1] = %+v, want pending adversarial-review", stages[1])
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
	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

	promoted, _, err := consumeOrdersNext(nextPath, ordersPath)
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

func TestNormalizeAndValidateOrdersDropsOrderWithMissingID(t *testing.T) {
	input := OrdersFile{
		Orders: []Order{
			{
				Title:  "missing id should be dropped",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Skill:    "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
			{
				ID:     "keep-1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Skill:    "execute",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
		},
	}

	got, changed, err := NormalizeAndValidateOrders(input, testLoopRegistry(), config.DefaultConfig())
	if err != nil {
		t.Fatalf("NormalizeAndValidateOrders: %v", err)
	}
	if !changed {
		t.Fatal("expected normalization to report changed=true")
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders len = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].ID != "keep-1" {
		t.Fatalf("remaining order ID = %q, want %q", got.Orders[0].ID, "keep-1")
	}
}

// --- Stage lifecycle function tests ---

func makeStage(status orderx.StageStatus) Stage {
	return Stage{
		TaskKey:  "execute",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Status:   status,
	}
}

func makeOrder(id string, status orderx.OrderStatus, stages []Stage) Order {
	return Order{
		ID:     id,
		Title:  "order " + id,
		Status: status,
		Stages: stages,
	}
}

func TestActiveStageForOrder(t *testing.T) {
	t.Run("returns first pending", func(t *testing.T) {
		order := makeOrder("1", OrderStatusActive, []Stage{
			makeStage(StageStatusCompleted),
			makeStage(StageStatusPending),
			makeStage(StageStatusPending),
		})
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
		})
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
		})
		idx, stage := activeStageForOrder(order)
		if idx != -1 || stage != nil {
			t.Errorf("expected (-1, nil), got (%d, %v)", idx, stage)
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
				}),
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
				makeOrder("keep", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("remove", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
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
			makeOrder("1", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
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
				}),
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
}

func TestFailStage(t *testing.T) {
	t.Run("marks current stage and order failed", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
					makeStage(StageStatusPending),
				}),
			},
		}

		of, err := failStage(of, "1", "test failure")
		if err != nil {
			t.Fatal(err)
		}
		if len(of.Orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(of.Orders))
		}
		if got := of.Orders[0].Status; got != OrderStatusFailed {
			t.Fatalf("order status = %q, want %q", got, OrderStatusFailed)
		}
		if got := of.Orders[0].Stages[1].Status; got != StageStatusFailed {
			t.Fatalf("stage[1] status = %q, want %q", got, StageStatusFailed)
		}
	})

	t.Run("missing order returns error", func(t *testing.T) {
		of := OrdersFile{}
		_, err := failStage(of, "nonexistent", "reason")
		if err == nil {
			t.Fatal("expected error for missing order")
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Run("mix of completed and pending", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("keep", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("cancel", OrderStatusActive, []Stage{
					makeStage(StageStatusCompleted),
					makeStage(StageStatusActive),
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
				}),
				makeOrder("b", OrderStatusActive, []Stage{
					makeStage(StageStatusPending),
				}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty)
		if len(candidates) != 2 {
			t.Fatalf("expected 2 candidates, got %d", len(candidates))
		}
		if candidates[0].OrderID != "a" || candidates[0].StageIndex != 1 {
			t.Errorf("candidate 0 = %+v", candidates[0])
		}
		if candidates[1].OrderID != "b" || candidates[1].StageIndex != 0 {
			t.Errorf("candidate 1 = %+v", candidates[1])
		}
	})

	t.Run("skips busy/adopted/ticketed and failed orders", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("busy", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("failed", OrderStatusFailed, []Stage{makeStage(StageStatusFailed)}),
				makeOrder("adopted", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("ticketed", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("free", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
			},
		}

		busySet := map[string]struct{}{"busy": {}}
		adoptedSet := map[string]struct{}{"adopted": {}}
		ticketedSet := map[string]struct{}{"ticketed": {}}

		candidates := dispatchableStages(of, busySet, adoptedSet, ticketedSet)
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
				}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty)
		if len(candidates) != 0 {
			t.Fatalf("expected 0 candidates (active stage = already dispatched), got %d", len(candidates))
		}
	})

	t.Run("skips degenerate empty stages", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("empty", OrderStatusActive, []Stage{}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty)
		if len(candidates) != 0 {
			t.Fatalf("expected 0 candidates for degenerate order, got %d", len(candidates))
		}
	})

	t.Run("respects array ordering", func(t *testing.T) {
		of := OrdersFile{
			Orders: []Order{
				makeOrder("first", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("second", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
				makeOrder("third", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
			},
		}

		candidates := dispatchableStages(of, empty, empty, empty)
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
			}),
			makeOrder("active-active", OrderStatusActive, []Stage{
				makeStage(StageStatusActive),
			}),
			makeOrder("done", OrderStatusCompleted, []Stage{
				makeStage(StageStatusCompleted),
			}),
		},
	}

	got := activeOrderIDs(of)
	want := []string{"active-pending", "active-active"}
	if len(got) != len(want) {
		t.Fatalf("activeOrderIDs len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("activeOrderIDs[%d] = %q, want %q", i, got[i], want[i])
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
			}),
			makeOrder("not-busy-main", OrderStatusActive, []Stage{
				makeStage(StageStatusCompleted),
				makeStage(StageStatusPending),
			}),
		},
	}

	busy := busyTargets(of)
	if !busy["busy-main"] {
		t.Fatal("expected busy-main to be busy")
	}
	if busy["not-busy-main"] {
		t.Fatal("did not expect not-busy-main to be busy")
	}
}

// Test value semantics — mutations should not affect original.
func TestLifecycleValueSemantics(t *testing.T) {
	original := OrdersFile{
		Orders: []Order{
			makeOrder("1", OrderStatusActive, []Stage{
				makeStage(StageStatusPending),
				makeStage(StageStatusPending),
			}),
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
						Runtime:  "process",
						Status:   StageStatusPending,
						Extra: map[string]json.RawMessage{
							"key": json.RawMessage(`"value"`),
						},
					},
				},
				Status: OrderStatusActive,
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
	if s.TaskKey != "execute" || s.Provider != "claude" || s.Runtime != "process" {
		t.Errorf("stage fields mismatch")
	}
	if string(s.Extra["key"]) != `"value"` {
		t.Errorf("Extra lost in clone round-trip")
	}
}
