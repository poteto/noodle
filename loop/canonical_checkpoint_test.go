package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestLoadOrBootstrapCanonicalFromLegacyState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "42",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Runtime: "process", Status: StageStatusActive},
					{TaskKey: "reflect", Skill: "reflect", Runtime: "process", Status: StageStatusPending},
				},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-review.json"), []byte(`{
  "items": [
    {
      "order_id": "42",
      "stage_index": 0,
      "task_key": "execute",
      "worktree_name": "42-0-execute",
      "session_id": "sess-42"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write pending review: %v", err)
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
	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.loadOrBootstrapCanonical(); err != nil {
		t.Fatalf("loadOrBootstrapCanonical: %v", err)
	}

	order := l.canonical.Orders["42"]
	if order.OrderID != "42" {
		t.Fatalf("bootstrapped order id = %q, want 42", order.OrderID)
	}
	if got := order.Stages[0].Status; got != state.StageReview {
		t.Fatalf("stage 0 status = %q, want %q", got, state.StageReview)
	}
	if len(order.Stages[0].Attempts) != 1 || order.Stages[0].Attempts[0].SessionID != "sess-42" {
		t.Fatalf("review attempt metadata missing: %+v", order.Stages[0].Attempts)
	}

	snapshot, err := reducer.ReadSnapshot(filepath.Join(runtimeDir, "state.snapshot.json"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if _, ok := snapshot.State.Orders["42"]; !ok {
		t.Fatalf("snapshot missing bootstrapped order: %+v", snapshot.State.Orders)
	}
}

func TestLoadCanonicalSnapshotRestoresEventCounter(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	snapshotPath := filepath.Join(runtimeDir, "state.snapshot.json")
	snapshot := reducer.DurableSnapshot{
		State: state.State{
			Orders:        map[string]state.OrderNode{},
			Mode:          state.RunModeAuto,
			SchemaVersion: statever.Current,
			LastEventID:   "7",
		},
		GeneratedAt: time.Now().UTC(),
	}
	if err := reducer.WriteSnapshotAtomic(snapshotPath, snapshot); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	loaded, err := l.loadCanonicalSnapshot()
	if err != nil {
		t.Fatalf("loadCanonicalSnapshot: %v", err)
	}
	if !loaded {
		t.Fatal("expected canonical snapshot to load")
	}

	l.emitEvent(ingest.EventModeChanged, map[string]any{"mode": string(state.RunModeManual)})
	if got := l.canonical.LastEventID; got != "8" {
		t.Fatalf("last event id after restore = %q, want 8", got)
	}

	restored, err := reducer.ReadSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("read restored snapshot: %v", err)
	}
	if restored.State.LastEventID != "8" {
		t.Fatalf("snapshot last event id = %q, want 8", restored.State.LastEventID)
	}
}

func TestCycleUsesInMemoryOrdersWithoutReload(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("seed empty orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise: &fakeMise{
			brief: mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "task", Status: "open"}}},
		},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	l.ordersLoaded = true
	l.orders = OrdersFile{
		Orders: []Order{
			{
				ID:     "memory-1",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Skill: "execute", Runtime: "process", Status: StageStatusPending}},
			},
		},
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(rt.calls) != 1 || rt.calls[0].Name == "" {
		t.Fatalf("expected in-memory order dispatch, got %+v", rt.calls)
	}
}
