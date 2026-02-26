package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/queuex"
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
	existing := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "existing-1", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	// Write next.
	next := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "new-1", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(nextPath, next); err != nil {
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
	got, err := queuex.ReadOrders(ordersPath)
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
	existing := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "dup", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
			{ID: "keep", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(ordersPath, existing); err != nil {
		t.Fatal(err)
	}

	// Write next with duplicate ID "dup" and new ID "new".
	next := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "dup", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
			{ID: "new", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	got, err := queuex.ReadOrders(ordersPath)
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
	merged := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "original", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
			{ID: "incoming", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(ordersPath, merged); err != nil {
		t.Fatal(err)
	}

	// orders-next.json still has the incoming order (crash replay).
	leftover := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "incoming", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(nextPath, leftover); err != nil {
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

	got, err := queuex.ReadOrders(ordersPath)
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
	next := queuex.OrdersFile{
		Orders: []queuex.Order{
			{ID: "first", Status: queuex.OrderStatusActive, Stages: []queuex.Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: queuex.StageStatusPending}}},
		},
	}
	if err := queuex.WriteOrdersAtomic(nextPath, next); err != nil {
		t.Fatal(err)
	}

	promoted, err := consumeOrdersNext(nextPath, ordersPath)
	if err != nil {
		t.Fatalf("consumeOrdersNext: %v", err)
	}
	if !promoted {
		t.Fatal("expected promoted=true")
	}

	got, err := queuex.ReadOrders(ordersPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Orders) != 1 || got.Orders[0].ID != "first" {
		t.Fatalf("expected 1 order with ID first, got: %+v", got.Orders)
	}
}

func TestOrdersFileConversionRoundTrip(t *testing.T) {
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

	roundTrip := fromOrdersFileX(toOrdersFileX(original))
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
		t.Errorf("Extra lost in conversion round-trip")
	}
	if len(o.OnFailure) != 1 || o.OnFailure[0].Prompt != "rollback" {
		t.Errorf("OnFailure mismatch")
	}
}
