package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestParkPendingReviewWritesFile(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
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
	})

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "42", stage: Stage{TaskKey: "execute", Skill: "execute"}},
		session:      &adoptedSession{id: "sess-42", status: "completed"},
		worktreeName: "42",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42"),
	}
	if err := l.parkPendingReview(cook, ""); err != nil {
		t.Fatalf("park pending review: %v", err)
	}

	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("read pending review file: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("pending review items = %d, want 1", len(items))
	}
	if items[0].OrderID != "42" {
		t.Fatalf("item order_id = %q, want 42", items[0].OrderID)
	}
	if items[0].SessionID != "sess-42" {
		t.Fatalf("session id = %q, want sess-42", items[0].SessionID)
	}
}

func TestLoadPendingReviewHydratesLoopState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	payload := `{
  "items": [
    {
      "order_id": "42",
      "task_key": "execute",
      "title": "Implement fix",
      "worktree_name": "42",
      "worktree_path": "` + filepath.Join(projectDir, ".worktrees", "42") + `",
      "session_id": "sess-42"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-review.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write pending review: %v", err)
	}

	// Write an order matching the pending review so reconciliation doesn't prune it.
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive}}},
	}}); err != nil {
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
	})

	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(l.cooks.pendingReview) != 1 {
		t.Fatalf("pendingReview size = %d, want 1", len(l.cooks.pendingReview))
	}
	pending, ok := l.cooks.pendingReview["42"]
	if !ok {
		t.Fatal("expected item 42 in pending review")
	}
	if pending.sessionID != "sess-42" {
		t.Fatalf("session id = %q, want sess-42", pending.sessionID)
	}
}

func TestPlanCycleSpawnsSkipsPendingReviewTargets(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{Orders: []Order{
		{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending}}},
		{ID: "43", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending}}},
	}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 2
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.cooks.pendingReview["42"] = &pendingReviewCook{cookIdentity: cookIdentity{orderID: "42"}}

	plan := l.planCycleSpawns(orders, mise.Brief{}, l.config.Concurrency.MaxCooks)
	if len(plan) != 1 || plan[0].OrderID != "43" {
		t.Fatalf("spawn plan = %#v, want only 43", plan)
	}
}
