package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

func BenchmarkPlanCycleSpawnsScale(b *testing.B) {
	sizes := []int{10, 100, 500, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("orders_%d", size), func(b *testing.B) {
			orders := OrdersFile{Orders: make([]Order, 0, size)}
			for i := 0; i < size; i++ {
				orders.Orders = append(orders.Orders, testOrder(fmt.Sprintf("bench-%d", i), "execute", "execute", "claude", "claude-opus-4-6"))
			}

			l := New(b.TempDir(), "noodle", config.DefaultConfig(), Dependencies{
				Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
				Worktree: &fakeWorktree{},
				Adapter:  &fakeAdapterRunner{},
				Mise:     &fakeMise{},
				Monitor:  fakeMonitor{},
				Registry: testLoopRegistry(),
			})

			brief := mise.Brief{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = l.planCycleSpawns(orders, brief, size)
			}
		})
	}
}

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
		b.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: make([]Order, 0, orderCount)}
	for i := 0; i < orderCount; i++ {
		orders.Orders = append(orders.Orders, testOrder(fmt.Sprintf("bench-cycle-%d", i), "execute", "execute", "claude", "claude-opus-4-6"))
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		b.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxConcurrency = orderCount
	cfg.Runtime.Process.MaxConcurrent = orderCount

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

	b.ResetTimer()
	if err := l.Cycle(context.Background()); err != nil {
		b.Fatalf("cycle: %v", err)
	}
}
