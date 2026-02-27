package loop

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/mise"
	nrt "github.com/poteto/noodle/runtime"
)

// mockSession implements nrt.SessionHandle with controllable lifecycle.
type mockSession struct {
	id     string
	status string
	done   chan struct{}
	mu     sync.Mutex
}

func newMockSession(id string) *mockSession {
	return &mockSession{id: id, status: "running", done: make(chan struct{})}
}

func (s *mockSession) ID() string { return s.id }
func (s *mockSession) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}
func (s *mockSession) Done() <-chan struct{} { return s.done }
func (s *mockSession) TotalCost() float64    { return 0 }
func (s *mockSession) Kill() error {
	s.mu.Lock()
	s.status = "killed"
	s.mu.Unlock()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}
func (s *mockSession) complete(status string) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

// mockRuntime implements nrt.Runtime for testing.
type mockRuntime struct {
	mu       sync.Mutex
	sessions []*mockSession
	health   chan nrt.HealthEvent
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		health: make(chan nrt.HealthEvent, 256),
	}
}

func (r *mockRuntime) Start(_ context.Context) error { return nil }
func (r *mockRuntime) Dispatch(_ context.Context, req dispatcher.DispatchRequest) (nrt.SessionHandle, error) {
	s := newMockSession(req.Name + "-id")
	r.mu.Lock()
	r.sessions = append(r.sessions, s)
	r.mu.Unlock()
	return s, nil
}
func (r *mockRuntime) Kill(h nrt.SessionHandle) error { return h.Kill() }
func (r *mockRuntime) Recover(_ context.Context) ([]nrt.RecoveredSession, error) {
	return nil, nil
}
func (r *mockRuntime) Health() <-chan nrt.HealthEvent { return r.health }
func (r *mockRuntime) Close() error                  { return nil }
func (r *mockRuntime) completeSessions(status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		s.complete(status)
	}
}

var _ nrt.Runtime = (*mockRuntime)(nil)

// TestScaleThroughput dispatches many orders, completes them, and verifies
// all advance correctly.
func TestScaleThroughput(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	orders := make([]Order, 20)
	for i := range orders {
		id := "order-" + time.Now().Format("150405") + "-" + string(rune('a'+i))
		orders[i] = testOrder(id, "execute", "execute", "claude", "claude-opus-4-6")
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: orders}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 20

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

	// Cycle 1: dispatch all 20 orders.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch cycle: %v", err)
	}
	if len(l.activeCooksByOrder) != 20 {
		t.Fatalf("active cooks = %d, want 20", len(l.activeCooksByOrder))
	}

	// Complete all sessions.
	mr.completeSessions("completed")
	time.Sleep(5 * time.Millisecond)

	// Cycle 2: drain all completions. A schedule order bootstraps after
	// all work orders complete, so 0 or 1 active cooks is correct.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	if len(l.activeCooksByOrder) > 1 {
		t.Fatalf("active cooks after completion = %d, want 0 or 1 (schedule)", len(l.activeCooksByOrder))
	}
}

// TestScaleBurstCompletions verifies exact-once processing when many
// sessions complete simultaneously.
func TestScaleBurstCompletions(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	n := 50
	orders := make([]Order, n)
	for i := range orders {
		orders[i] = testOrder("burst-"+string(rune('A'+i%26))+string(rune('0'+i/26)), "execute", "execute", "claude", "opus")
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: orders}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = n

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

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch cycle: %v", err)
	}
	if len(l.activeCooksByOrder) != n {
		t.Fatalf("dispatched %d, want %d", len(l.activeCooksByOrder), n)
	}

	// Complete all at once.
	mr.completeSessions("completed")
	time.Sleep(5 * time.Millisecond)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("burst cycle: %v", err)
	}
	if len(l.activeCooksByOrder) > 1 {
		t.Fatalf("remaining active = %d, want 0 or 1 (schedule)", len(l.activeCooksByOrder))
	}
}

// TestScaleLoopStateSnapshot verifies State() returns correct data under load.
func TestScaleLoopStateSnapshot(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		testOrder("snap-1", "execute", "execute", "claude", "opus"),
		testOrder("snap-2", "execute", "execute", "claude", "opus"),
	}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	mr := newMockRuntime()
	runtimes := nrt.NewRuntimeMap()
	runtimes.Register("tmux", mr)

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: runtimes,
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	state := l.State()
	if state.LoopState != "running" {
		t.Fatalf("loop state = %q, want running", state.LoopState)
	}
	if len(state.ActiveCooks) != 2 {
		t.Fatalf("active cooks = %d, want 2", len(state.ActiveCooks))
	}
	if state.ActiveSummary.Total != 2 {
		t.Fatalf("summary total = %d, want 2", state.ActiveSummary.Total)
	}
}
