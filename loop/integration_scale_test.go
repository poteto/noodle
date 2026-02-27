package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
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

	sp := &fakeDispatcher{}
	wt := &fakeWorktree{}
	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = orderCount
	cfg.Concurrency.MergeBackpressureThreshold = orderCount * 2

	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: sp,
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
	if len(sp.sessions) != orderCount {
		t.Fatalf("dispatched sessions = %d, want %d", len(sp.sessions), orderCount)
	}

	for _, session := range sp.sessions {
		session.status = "completed"
		close(session.done)
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
