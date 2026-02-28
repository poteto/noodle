package loop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestControlAdvanceAdvancesBlockedStage(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusCompleted},
			{TaskKey: "quality", Status: StageStatusPending},
		},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.controlAdvance("42"); err != nil {
		t.Fatalf("controlAdvance: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	// After advancing the pending stage, order should be removed (final stage).
	for _, o := range orders.Orders {
		if o.ID == "42" {
			t.Fatal("order 42 should be removed after advancing final stage")
		}
	}
}

func TestControlAdvanceMissingOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	err := l.controlAdvance("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing order")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %q, want 'not found'", err.Error())
	}
}

func TestControlAddStageInsertsAfterLastCompleted(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusCompleted},
			{TaskKey: "quality", Status: StageStatusPending},
		},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	cmd := ControlCommand{
		Action:   "add-stage",
		OrderID:  "42",
		TaskKey:  "oops",
		Prompt:   "Fix test failures",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Skill:    "oops",
	}
	if err := l.controlAddStage(cmd); err != nil {
		t.Fatalf("controlAddStage: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	stages := orders.Orders[0].Stages
	if len(stages) != 3 {
		t.Fatalf("stages count = %d, want 3", len(stages))
	}
	// New stage should be inserted after the completed stage (index 1).
	if stages[1].TaskKey != "oops" {
		t.Fatalf("stages[1].TaskKey = %q, want oops", stages[1].TaskKey)
	}
	if stages[1].Status != StageStatusPending {
		t.Fatalf("stages[1].Status = %q, want pending", stages[1].Status)
	}
	if stages[1].Prompt != "Fix test failures" {
		t.Fatalf("stages[1].Prompt = %q, want 'Fix test failures'", stages[1].Prompt)
	}
	// Original quality stage should be pushed to index 2.
	if stages[2].TaskKey != "quality" {
		t.Fatalf("stages[2].TaskKey = %q, want quality", stages[2].TaskKey)
	}
}

func TestControlParkReviewCreatesReview(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.controlParkReview("42", "needs human review"); err != nil {
		t.Fatalf("controlParkReview: %v", err)
	}

	pending, ok := l.cooks.pendingReview["42"]
	if !ok {
		t.Fatal("expected pending review for order 42")
	}
	if pending.reason != "needs human review" {
		t.Fatalf("reason = %q, want 'needs human review'", pending.reason)
	}
	if pending.orderID != "42" {
		t.Fatalf("orderID = %q, want 42", pending.orderID)
	}

	// Verify persisted to disk.
	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("read pending review: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("pending review items = %d, want 1", len(items))
	}
	if items[0].OrderID != "42" {
		t.Fatalf("persisted order_id = %q, want 42", items[0].OrderID)
	}
}
