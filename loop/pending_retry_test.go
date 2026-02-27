package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestWritePendingRetryPersistsFile(t *testing.T) {
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
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
	})

	l.cooks.pendingRetry["42"] = &pendingRetryCook{
		cookIdentity: cookIdentity{
			orderID:    "42",
			stageIndex: 0,
			stage:      Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "opus"},
			plan:       []string{"plans/42/overview"},
		},
		isOnFailure: false,
		orderStatus: OrderStatusActive,
		attempt:     2,
		displayName: "chef-bravo",
	}

	if err := l.writePendingRetry(); err != nil {
		t.Fatalf("writePendingRetry: %v", err)
	}

	items, err := readPendingRetryFile(runtimeDir)
	if err != nil {
		t.Fatalf("readPendingRetryFile: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].OrderID != "42" {
		t.Fatalf("order_id = %q, want 42", items[0].OrderID)
	}
	if items[0].Attempt != 2 {
		t.Fatalf("attempt = %d, want 2", items[0].Attempt)
	}
	if items[0].DisplayName != "chef-bravo" {
		t.Fatalf("display_name = %q, want chef-bravo", items[0].DisplayName)
	}
}

func TestLoadPendingRetryHydratesState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	payload := `{
  "items": [
    {
      "order_id": "42",
      "stage_index": 1,
      "task_key": "execute",
      "skill": "execute",
      "provider": "claude",
      "model": "opus",
      "is_on_failure": false,
      "order_status": "active",
      "plan": ["plans/42/overview"],
      "attempt": 3,
      "display_name": "chef-alpha"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-retry.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write pending-retry.json: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
	})

	if err := l.loadPendingRetry(); err != nil {
		t.Fatalf("loadPendingRetry: %v", err)
	}
	if len(l.cooks.pendingRetry) != 1 {
		t.Fatalf("pendingRetry size = %d, want 1", len(l.cooks.pendingRetry))
	}
	p, ok := l.cooks.pendingRetry["42"]
	if !ok {
		t.Fatal("expected item 42 in pendingRetry")
	}
	if p.attempt != 3 {
		t.Fatalf("attempt = %d, want 3", p.attempt)
	}
	if p.displayName != "chef-alpha" {
		t.Fatalf("displayName = %q, want chef-alpha", p.displayName)
	}
	if p.stage.TaskKey != "execute" {
		t.Fatalf("task_key = %q, want execute", p.stage.TaskKey)
	}
}

func TestReconcilePendingRetryPrunesRemovedOrders(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	// Write orders with only order "43" — "42" is gone.
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "43", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusPending}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	l.cooks.pendingRetry["42"] = &pendingRetryCook{cookIdentity: cookIdentity{orderID: "42", stage: Stage{TaskKey: "execute"}}, attempt: 1}
	l.cooks.pendingRetry["43"] = &pendingRetryCook{cookIdentity: cookIdentity{orderID: "43", stage: Stage{TaskKey: "execute"}}, attempt: 1}

	if err := l.reconcilePendingRetry(); err != nil {
		t.Fatalf("reconcilePendingRetry: %v", err)
	}

	if _, ok := l.cooks.pendingRetry["42"]; ok {
		t.Fatal("expected item 42 to be pruned (order removed)")
	}
	if _, ok := l.cooks.pendingRetry["43"]; !ok {
		t.Fatal("expected item 43 to be kept (order still exists)")
	}
}

func TestReconcilePendingRetryPrunesAdoptedSessions(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusActive}}},
		{ID: "43", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusPending}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Simulate that order "42" was recovered with a live session.
	l.cooks.adoptedTargets["42"] = "sess-42"
	l.cooks.pendingRetry["42"] = &pendingRetryCook{cookIdentity: cookIdentity{orderID: "42", stage: Stage{TaskKey: "execute"}}, attempt: 1}
	l.cooks.pendingRetry["43"] = &pendingRetryCook{cookIdentity: cookIdentity{orderID: "43", stage: Stage{TaskKey: "execute"}}, attempt: 1}

	if err := l.reconcilePendingRetry(); err != nil {
		t.Fatalf("reconcilePendingRetry: %v", err)
	}

	if _, ok := l.cooks.pendingRetry["42"]; ok {
		t.Fatal("expected item 42 to be pruned (adopted session)")
	}
	if _, ok := l.cooks.pendingRetry["43"]; !ok {
		t.Fatal("expected item 43 to be kept (no adopted session)")
	}
}

func TestLoadPendingRetryReconcileDuringStartup(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	// Order 42 has an active stage and a pending retry on disk.
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusActive}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	retryPayload := `{
  "items": [
    {
      "order_id": "42",
      "stage_index": 0,
      "task_key": "execute",
      "order_status": "active",
      "attempt": 2,
      "display_name": "chef-alpha"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-retry.json"), []byte(retryPayload), 0o644); err != nil {
		t.Fatalf("write pending-retry.json: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}

	// reconcile() will load pending retries after building the live-session index.
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// No adopted sessions, so retry should survive.
	if _, ok := l.cooks.pendingRetry["42"]; !ok {
		t.Fatal("expected item 42 in pendingRetry after startup reconcile")
	}
	if l.cooks.pendingRetry["42"].attempt != 2 {
		t.Fatalf("attempt = %d, want 2", l.cooks.pendingRetry["42"].attempt)
	}
}

func TestLoadPendingRetryCorruptFileStartsFresh(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-retry.json"), []byte("not valid json{{{"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
	})

	if err := l.loadPendingRetry(); err != nil {
		t.Fatalf("loadPendingRetry should not error on corrupt file: %v", err)
	}
	if len(l.cooks.pendingRetry) != 0 {
		t.Fatalf("pendingRetry size = %d, want 0 (fresh start)", len(l.cooks.pendingRetry))
	}
}
