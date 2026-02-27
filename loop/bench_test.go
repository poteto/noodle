package loop

import (
	"fmt"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
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
				Dispatcher: &fakeDispatcher{},
				Worktree:   &fakeWorktree{},
				Adapter:    &fakeAdapterRunner{},
				Mise:       &fakeMise{},
				Monitor:    fakeMonitor{},
				Registry:   testLoopRegistry(),
			})

			brief := mise.Brief{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = l.planCycleSpawns(orders, brief, size)
			}
		})
	}
}
