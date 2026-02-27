package loop

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	nrt "github.com/poteto/noodle/runtime"
)

func BenchmarkCycleDispatch(b *testing.B) {
	for _, count := range []int{10, 50, 100} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchCycleDispatch(b, count)
			}
		})
	}
}

func benchCycleDispatch(b *testing.B, orderCount int) {
	b.Helper()
	projectDir := b.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		b.Fatalf("mkdir: %v", err)
	}

	orders := make([]Order, orderCount)
	for i := range orders {
		orders[i] = testOrder("bench-"+strconv.Itoa(i), "execute", "execute", "claude", "opus")
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: orders}); err != nil {
		b.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = orderCount

	mr := newMockRuntime()
	runtimes := nrt.NewRuntimeMap()
	runtimes.Register("tmux", mr)

	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: runtimes,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
	})

	b.ResetTimer()
	if err := l.Cycle(context.Background()); err != nil {
		b.Fatalf("cycle: %v", err)
	}
}
