package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestScaleBurstCompletionProcessesAllOrders(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	const orderCount = 100
	orders := OrdersFile{Orders: make([]Order, 0, orderCount)}
	for i := 0; i < orderCount; i++ {
		id := fmt.Sprintf("scale-%03d", i)
		orders.Orders = append(orders.Orders, testOrder(id, "execute", "execute", "claude", "claude-opus-4-6"))
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = orderCount
	cfg.Concurrency.MergeBackpressureThreshold = orderCount * 2
	cfg.Runtime.Tmux.MaxConcurrent = orderCount

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch cycle: %v", err)
	}
	if len(rt.sessions) != orderCount {
		t.Fatalf("dispatched sessions = %d, want %d", len(rt.sessions), orderCount)
	}

	for _, session := range rt.sessions {
		session.complete("completed")
	}

	for i := 0; i < 20; i++ {
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("completion cycle %d: %v", i+1, err)
		}
		current, err := readOrders(ordersPath)
		if err != nil {
			t.Fatalf("read orders: %v", err)
		}
		if len(current.Orders) == 0 {
			return
		}
	}

	current, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	t.Fatalf("orders remaining after burst completion: %d", len(current.Orders))
}

func TestScaleLoopStateSnapshotIncludesActiveSummary(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		testOrder("snap-1", "execute", "execute", "claude", "claude-opus-4-6"),
		testOrder("snap-2", "execute", "execute", "claude", "claude-opus-4-6"),
	}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	state := l.State()
	if state.Status != string(StateRunning) {
		t.Fatalf("state status = %q, want %q", state.Status, StateRunning)
	}
	if len(state.ActiveCooks) != 2 {
		t.Fatalf("active cooks = %d, want 2", len(state.ActiveCooks))
	}
	if state.ActiveSummary.Total != 2 {
		t.Fatalf("active summary total = %d, want 2", state.ActiveSummary.Total)
	}
}

// TestMockRuntimeDispatchAndComplete verifies the full dispatch→complete
// lifecycle using MockRuntime through the Runtimes map path.
func TestMockRuntimeDispatchAndComplete(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		testOrder("rt-1", "execute", "execute", "claude", "claude-opus-4-6"),
		testOrder("rt-2", "execute", "execute", "claude", "claude-opus-4-6"),
	}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch cycle: %v", err)
	}
	if len(rt.sessions) != 2 {
		t.Fatalf("dispatched sessions = %d, want 2", len(rt.sessions))
	}

	// Complete all sessions.
	for _, s := range rt.sessions {
		s.complete("completed")
	}

	for i := 0; i < 10; i++ {
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("completion cycle %d: %v", i+1, err)
		}
		current, err := readOrders(ordersPath)
		if err != nil {
			t.Fatalf("read orders: %v", err)
		}
		if len(current.Orders) == 0 {
			return
		}
	}

	current, _ := readOrders(ordersPath)
	t.Fatalf("orders remaining: %d", len(current.Orders))
}

// TestMockRuntimeRecoverBuildsAdoptedIndex verifies that Runtime.Recover()
// results are used to build the adopted session index during reconcile.
func TestMockRuntimeRecoverBuildsAdoptedIndex(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "order-1", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusActive}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.recovered = []loopruntime.RecoveredSession{
		{
			OrderID:       "order-1",
			SessionHandle: &mockSession{id: "sess-1", status: "running", done: make(chan struct{})},
			RuntimeName:   "tmux",
			Reason:        "test recovery",
		},
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if sid, ok := l.adoptedTargets["order-1"]; !ok {
		t.Fatal("expected order-1 in adoptedTargets")
	} else if sid != "sess-1" {
		t.Fatalf("adoptedTargets[order-1] = %q, want sess-1", sid)
	}
	if len(l.adoptedSessions) != 1 {
		t.Fatalf("adoptedSessions = %d, want 1", len(l.adoptedSessions))
	}
}

// TestMockRuntimeScaleBurstViaRuntimes mirrors TestScaleBurstCompletionProcessesAllOrders
// but dispatches through the Runtimes map instead of the legacy Dispatcher fallback.
func TestMockRuntimeScaleBurstViaRuntimes(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	const orderCount = 50
	orders := OrdersFile{Orders: make([]Order, 0, orderCount)}
	for i := 0; i < orderCount; i++ {
		id := fmt.Sprintf("rt-scale-%03d", i)
		orders.Orders = append(orders.Orders, testOrder(id, "execute", "execute", "claude", "claude-opus-4-6"))
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = orderCount
	cfg.Concurrency.MergeBackpressureThreshold = orderCount * 2

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch cycle: %v", err)
	}
	if len(rt.sessions) != orderCount {
		t.Fatalf("dispatched sessions = %d, want %d", len(rt.sessions), orderCount)
	}

	for _, s := range rt.sessions {
		s.complete("completed")
	}

	for i := 0; i < 20; i++ {
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("completion cycle %d: %v", i+1, err)
		}
		current, err := readOrders(ordersPath)
		if err != nil {
			t.Fatalf("read orders: %v", err)
		}
		if len(current.Orders) == 0 {
			return
		}
	}

	current, _ := readOrders(ordersPath)
	t.Fatalf("orders remaining after burst: %d", len(current.Orders))
}
