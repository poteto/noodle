package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestReconcileFailedOrdersArchivesAndSummarizes(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "schedule",
				Title:  "scheduling tasks",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "schedule", Status: StageStatusPending}},
			},
			{
				ID:     "abc-123",
				Title:  "fix auth bug",
				Status: OrderStatusFailed,
				Stages: []Stage{
					{TaskKey: "execute", Status: StageStatusFailed},
					{TaskKey: "quality", Status: StageStatusPending},
				},
			},
			{
				ID:     "def-456",
				Title:  "add logging",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusActive}},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
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

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcileFailedOrders(); err != nil {
		t.Fatalf("reconcileFailedOrders: %v", err)
	}

	// Failed order should be removed from orders.json.
	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	for _, o := range updated.Orders {
		if o.ID == "abc-123" {
			t.Fatal("failed order abc-123 should have been removed from orders.json")
		}
	}
	if len(updated.Orders) != 2 {
		t.Fatalf("expected 2 orders remaining, got %d", len(updated.Orders))
	}

	// Reconciled failures should be populated.
	if len(l.reconciledFailures) != 1 {
		t.Fatalf("expected 1 reconciled failure, got %d", len(l.reconciledFailures))
	}
	f := l.reconciledFailures[0]
	if f.OrderID != "abc-123" {
		t.Fatalf("failure OrderID = %q, want %q", f.OrderID, "abc-123")
	}
	if f.Title != "fix auth bug" {
		t.Fatalf("failure Title = %q, want %q", f.Title, "fix auth bug")
	}
	if f.TaskKey != "execute" {
		t.Fatalf("failure TaskKey = %q, want %q", f.TaskKey, "execute")
	}
}

func TestReconcileFailedOrdersSkipsScheduleOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "schedule",
				Title:  "scheduling tasks",
				Status: OrderStatusFailed,
				Stages: []Stage{{TaskKey: "schedule", Status: StageStatusFailed}},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
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

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcileFailedOrders(); err != nil {
		t.Fatalf("reconcileFailedOrders: %v", err)
	}

	// Schedule order should not be archived.
	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 {
		t.Fatalf("expected 1 order remaining, got %d", len(updated.Orders))
	}
	if len(l.reconciledFailures) != 0 {
		t.Fatalf("expected 0 reconciled failures, got %d", len(l.reconciledFailures))
	}
}

func TestReconcileFailedOrdersNoopWhenNoFailures(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "active-order",
				Title:  "doing stuff",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
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

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcileFailedOrders(); err != nil {
		t.Fatalf("reconcileFailedOrders: %v", err)
	}

	if len(l.reconciledFailures) != 0 {
		t.Fatalf("expected 0 reconciled failures, got %d", len(l.reconciledFailures))
	}
}
