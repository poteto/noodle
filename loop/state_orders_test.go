package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/statever"
)

func TestMutateOrdersStateSkipsWriteWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")

	seed := OrdersFile{
		GeneratedAt: time.Now().Truncate(time.Second),
		Orders: []Order{
			makeOrder("1", OrderStatusActive, []Stage{
				makeStage(StageStatusPending),
			}),
		},
	}
	if err := writeOrdersAtomic(ordersPath, seed); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	l := &Loop{
		deps: Dependencies{OrdersFile: ordersPath},
	}

	// Load orders so currentOrders() doesn't need to read from disk.
	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}

	// Record the file's mtime before the no-op mutate.
	infoBefore, err := os.Stat(ordersPath)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	mtimeBefore := infoBefore.ModTime()

	// Sleep briefly so any write would produce a different mtime.
	time.Sleep(50 * time.Millisecond)

	// Call mutateOrdersState with a mutator that returns changed=false.
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		return false, nil
	}); err != nil {
		t.Fatalf("mutateOrdersState: %v", err)
	}

	// Verify the file was NOT written (mtime unchanged).
	infoAfter, err := os.Stat(ordersPath)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if !infoAfter.ModTime().Equal(mtimeBefore) {
		t.Fatalf("file was written despite changed=false: mtime before=%v, after=%v",
			mtimeBefore, infoAfter.ModTime())
	}
}

func TestMutateOrdersStateWritesWhenChanged(t *testing.T) {
	dir := t.TempDir()
	ordersPath := filepath.Join(dir, "orders.json")

	seed := OrdersFile{
		GeneratedAt: time.Now().Truncate(time.Second),
		Orders: []Order{
			makeOrder("1", OrderStatusActive, []Stage{
				makeStage(StageStatusPending),
			}),
		},
	}
	if err := writeOrdersAtomic(ordersPath, seed); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	l := &Loop{
		deps: Dependencies{OrdersFile: ordersPath},
	}
	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}

	// Mutate with changed=true.
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		orders.Orders[0].Title = "updated"
		return true, nil
	}); err != nil {
		t.Fatalf("mutateOrdersState: %v", err)
	}

	// Verify the mutation was persisted.
	got, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("readOrders: %v", err)
	}
	if got.Orders[0].Title != "updated" {
		t.Fatalf("title = %q, want %q", got.Orders[0].Title, "updated")
	}
}

func TestFlushStateWritesStateMarker(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)

	l := &Loop{
		runtimeDir: dir,
		deps: Dependencies{
			OrdersFile: filepath.Join(dir, "orders.json"),
			Now:        func() time.Time { return now },
		},
		ordersLoaded: true,
		orders: OrdersFile{
			GeneratedAt: now,
			Orders: []Order{
				makeOrder("1", OrderStatusActive, []Stage{makeStage(StageStatusPending)}),
			},
		},
		cooks: cookTracker{pendingReview: map[string]*pendingReviewCook{}},
	}

	if err := l.flushState(); err != nil {
		t.Fatalf("flushState: %v", err)
	}

	marker, err := statever.Read(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("read state marker: %v", err)
	}
	if marker.SchemaVersion != statever.Current {
		t.Fatalf("schema version = %d, want %d", marker.SchemaVersion, statever.Current)
	}
	if !marker.GeneratedAt.Equal(now) {
		t.Fatalf("generated_at = %v, want %v", marker.GeneratedAt, now)
	}
}
