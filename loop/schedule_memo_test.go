package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

// scheduleMemoEnv drives full loop cycles against a controllable clock and a
// mutable mise brief, so tests can assert how many schedule sessions the loop
// spawns as decision-relevant state moves (or doesn't).
type scheduleMemoEnv struct {
	loop           *Loop
	rt             *mockRuntime
	mise           *fakeMise
	clock          time.Time
	projectDir     string
	ordersPath     string
	ordersNextPath string
}

func newScheduleMemoEnv(t *testing.T, backlog ...adapter.BacklogItem) *scheduleMemoEnv {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	env := &scheduleMemoEnv{
		rt:             newMockRuntime(),
		mise:           &fakeMise{brief: mise.Brief{Backlog: backlog}},
		clock:          time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		projectDir:     projectDir,
		ordersPath:     ordersPath,
		ordersNextPath: filepath.Join(runtimeDir, "orders-next.json"),
	}
	env.loop = env.newLoop(t)
	return env
}

// newLoop builds a loop over the same project directory. Calling it a second
// time models a process restart: on-disk state survives, in-memory memo does not.
func (e *scheduleMemoEnv) newLoop(t *testing.T) *Loop {
	t.Helper()
	l := New(e.projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": e.rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       e.mise,
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        func() time.Time { return e.clock },
		OrdersFile: e.ordersPath,
	})
	return l
}

func (e *scheduleMemoEnv) cycle(t *testing.T) {
	t.Helper()
	if err := e.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
}

func (e *scheduleMemoEnv) cycles(t *testing.T, count int, step time.Duration) {
	t.Helper()
	for i := 0; i < count; i++ {
		e.clock = e.clock.Add(step)
		e.cycle(t)
	}
}

func (e *scheduleMemoEnv) scheduleSpawns() int {
	count := 0
	for _, call := range e.rt.calls {
		if call.Name == scheduleOrderID {
			count++
		}
	}
	return count
}

// completeSchedule simulates the schedule session writing the given
// orders-next.json payload and exiting cleanly.
func (e *scheduleMemoEnv) completeSchedule(t *testing.T, ordersNext string) {
	t.Helper()
	if err := os.WriteFile(e.ordersNextPath, []byte(ordersNext), 0o644); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}
	for _, session := range e.rt.sessions {
		if session.Status() == "running" {
			session.complete("completed")
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.loop.watcherCount.Load() > 0 {
		if time.Now().After(deadline) {
			t.Fatal("session watchers still running after completion")
		}
		time.Sleep(time.Millisecond)
	}
	e.cycle(t)
}

const emptyOrdersNext = `{"orders":[]}`

// settle runs one dispatch → empty decision → promotion round trip and returns
// with the empty decision memoized.
func (e *scheduleMemoEnv) settleWithEmptyDecision(t *testing.T) {
	t.Helper()
	e.cycle(t)
	if got := e.scheduleSpawns(); got != 1 {
		t.Fatalf("schedule spawns after first cycle = %d, want 1", got)
	}
	e.completeSchedule(t, emptyOrdersNext)
	if got := e.scheduleSpawns(); got != 1 {
		t.Fatalf("schedule spawns after empty promotion = %d, want 1", got)
	}
}

func TestEmptyScheduleDecisionSuppressesRespawn(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.settleWithEmptyDecision(t)

	// Five minutes was the old cooldown window; cross several of them.
	env.cycles(t, 6, 6*time.Minute)

	if got := env.scheduleSpawns(); got != 1 {
		t.Fatalf("schedule spawns across cooldown windows = %d, want 1", got)
	}
	if env.loop.state != StateIdle {
		t.Fatalf("loop state = %q, want idle", env.loop.state)
	}
}

func TestScheduleLifecycleChurnDoesNotInvalidateEmptyDecision(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.settleWithEmptyDecision(t)

	// Exactly the state the schedule cycle mutates by running: its own
	// session in the active summary and history, and its lifecycle events.
	brief := env.mise.brief
	brief.GeneratedAt = env.clock
	brief.ActiveSummary = mise.ActiveSummary{Total: 1, ByTaskKey: map[string]int{"schedule": 1}}
	brief.Resources = mise.ResourceSnapshot{MaxConcurrency: 4, Active: 1, Available: 3}
	brief.RecentHistory = []mise.HistoryItem{{SessionID: "schedule-id", TaskKey: "schedule", Status: "completed"}}
	brief.RecentEvents = []mise.RecentEvent{
		{Type: "schedule.completed", Seq: 3, At: env.clock, Summary: "order schedule completed"},
		{Type: "order.completed", Seq: 4, At: env.clock, Summary: "order schedule completed"},
	}
	env.mise.brief = brief

	env.cycles(t, 3, time.Minute)

	if got := env.scheduleSpawns(); got != 1 {
		t.Fatalf("schedule spawns after lifecycle churn = %d, want 1", got)
	}
}

func TestBacklogChangeReschedulesExactlyOnce(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.settleWithEmptyDecision(t)

	env.mise.brief = mise.Brief{Backlog: []adapter.BacklogItem{
		{ID: "1", Title: "fix login", Status: "open"},
		{ID: "2", Title: "add metrics", Status: "open"},
	}}
	env.cycles(t, 1, time.Minute)
	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after backlog change = %d, want 2", got)
	}

	// The new digest is in flight, not decided: no second dispatch while the
	// scheduler works, and none after it promotes an empty result either.
	env.cycles(t, 3, time.Minute)
	env.completeSchedule(t, emptyOrdersNext)
	env.cycles(t, 3, 6*time.Minute)
	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after backlog change settled = %d, want 2", got)
	}
}

// A backlog change that lands while the scheduler is already running was not
// part of the decision it is about to promote, so it must still earn a
// dispatch of its own.
func TestBacklogChangeDuringScheduleSessionIsNotMemoized(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.cycle(t)
	if got := env.scheduleSpawns(); got != 1 {
		t.Fatalf("schedule spawns after first cycle = %d, want 1", got)
	}

	env.mise.brief = mise.Brief{Backlog: []adapter.BacklogItem{
		{ID: "1", Title: "fix login", Status: "open"},
		{ID: "2", Title: "add metrics", Status: "open"},
	}}
	env.completeSchedule(t, emptyOrdersNext)
	env.cycles(t, 2, time.Minute)

	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after mid-session backlog change = %d, want 2", got)
	}
}

func TestChefSteerInvalidatesEmptyDecision(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.settleWithEmptyDecision(t)

	if err := env.loop.rescheduleForChefPrompt("look at the metrics work"); err != nil {
		t.Fatalf("rescheduleForChefPrompt: %v", err)
	}
	env.cycles(t, 1, time.Minute)

	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after chef steer = %d, want 2", got)
	}
}

// Bounded restart exception: the memo is in-memory only, so a restart costs at
// most one extra schedule session before the loop is quiet again.
func TestRestartCostsExactlyOneScheduleDispatch(t *testing.T) {
	env := newScheduleMemoEnv(t, adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	env.settleWithEmptyDecision(t)

	env.loop = env.newLoop(t)
	env.cycles(t, 1, time.Minute)
	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after restart = %d, want 2", got)
	}

	env.completeSchedule(t, emptyOrdersNext)
	env.cycles(t, 4, 6*time.Minute)
	if got := env.scheduleSpawns(); got != 2 {
		t.Fatalf("schedule spawns after restart settled = %d, want 2", got)
	}
}
