package loop

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

// logEntry is a captured slog record for test assertions.
type logEntry struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

// capturingHandler is a slog.Handler that records log entries in memory.
// Safe for concurrent use (required by slog.Handler contract).
type capturingHandler struct {
	mu      sync.Mutex
	entries []logEntry
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	entry := logEntry{
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make(map[string]any),
	}
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	h.entries = append(h.entries, entry)
	h.mu.Unlock()
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingHandler) snapshot() []logEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]logEntry, len(h.entries))
	copy(out, h.entries)
	return out
}

// findEntry returns the first log entry matching the message.
func (h *capturingHandler) findEntry(msg string) (logEntry, bool) {
	for _, e := range h.snapshot() {
		if e.Message == msg {
			return e, true
		}
	}
	return logEntry{}, false
}

// hasMessage returns true if any entry matches the message.
func (h *capturingHandler) hasMessage(msg string) bool {
	_, ok := h.findEntry(msg)
	return ok
}

func (h *capturingHandler) countMessage(msg string) int {
	count := 0
	for _, e := range h.snapshot() {
		if e.Message == msg {
			count++
		}
	}
	return count
}

// newTestLogger creates a *slog.Logger backed by a capturingHandler.
func newTestLogger() (*slog.Logger, *capturingHandler) {
	h := &capturingHandler{}
	return slog.New(h), h
}

// newTestLoop creates a Loop wired with a capturing logger and common fakes.
func newTestLoop(t *testing.T, logger *slog.Logger, opts ...func(*testLoopOpts)) *testLoopContext {
	t.Helper()
	o := &testLoopOpts{}
	for _, fn := range opts {
		fn(o)
	}

	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	fm := &fakeMise{}
	if o.brief != nil {
		fm.brief = *o.brief
	}

	cfg := config.DefaultConfig()
	if o.maxRetries != nil {
		cfg.Recovery.MaxRetries = *o.maxRetries
	}

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    ar,
		Mise:       fm,
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Logger:     logger,
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	return &testLoopContext{
		loop:       l,
		runtime:    rt,
		worktree:   wt,
		adapter:    ar,
		mise:       fm,
		ordersPath: ordersPath,
		runtimeDir: runtimeDir,
		projectDir: projectDir,
	}
}

type testLoopOpts struct {
	brief      *mise.Brief
	maxRetries *int
}

type testLoopContext struct {
	loop       *Loop
	runtime    *mockRuntime
	worktree   *fakeWorktree
	adapter    *fakeAdapterRunner
	mise       *fakeMise
	ordersPath string
	runtimeDir string
	projectDir string
}

func TestLogDispatchCook(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	orders := OrdersFile{Orders: []Order{testOrder("item-1", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := writeOrdersAtomic(tc.ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	entry, ok := handler.findEntry("cook dispatched")
	if !ok {
		t.Fatal("expected 'cook dispatched' log entry")
	}
	if entry.Attrs["order"] != "item-1" {
		t.Fatalf("order attr = %v, want item-1", entry.Attrs["order"])
	}
	if entry.Attrs["session"] == nil || entry.Attrs["session"] == "" {
		t.Fatal("expected non-empty session attr")
	}
}

func TestLogDispatchSchedule(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
		o.brief = &brief
	})

	// Empty orders with plans triggers bootstrap schedule.
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if !handler.hasMessage("schedule dispatched") {
		t.Fatal("expected 'schedule dispatched' log entry")
	}
}

func TestLogCompletionMerge(t *testing.T) {
	logger, handler := newTestLogger()
	brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		o.brief = &brief
	})

	orders := OrdersFile{Orders: []Order{testOrder("item-1", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := writeOrdersAtomic(tc.ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(tc.runtime.sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(tc.runtime.sessions))
	}

	tc.runtime.sessions[0].complete("completed")

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}

	if !handler.hasMessage("cook completing") {
		t.Fatal("expected 'cook completing' log entry")
	}
	if !handler.hasMessage("cook merged") {
		t.Fatal("expected 'cook merged' log entry")
	}
}

func TestLogScheduleCompleted(t *testing.T) {
	logger, handler := newTestLogger()
	brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		o.brief = &brief
	})

	// Empty orders + plans → bootstrap schedule dispatch.
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(tc.runtime.sessions) < 1 {
		t.Fatal("expected schedule dispatch")
	}

	tc.runtime.sessions[0].complete("completed")

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}

	if !handler.hasMessage("schedule completed") {
		t.Fatal("expected 'schedule completed' log entry")
	}
}

func TestLogStateTransition(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	// Write a pause command.
	controlPath := filepath.Join(tc.runtimeDir, "control.ndjson")
	if err := os.WriteFile(controlPath, []byte(`{"id":"cmd-1","action":"pause"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	entry, ok := handler.findEntry("state changed")
	if !ok {
		t.Fatal("expected 'state changed' log entry")
	}
	if entry.Attrs["from"] != "running" {
		t.Fatalf("from = %v, want running", entry.Attrs["from"])
	}
	if entry.Attrs["to"] != "paused" {
		t.Fatalf("to = %v, want paused", entry.Attrs["to"])
	}
}

func TestLogSetStateNoOp(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	// Loop starts in StateRunning. Calling setState(StateRunning) should be a no-op.
	tc.loop.setState(StateRunning)

	if handler.hasMessage("state changed") {
		t.Fatal("setState should not log when state is unchanged")
	}
}

func TestLogControlCommand(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	controlPath := filepath.Join(tc.runtimeDir, "control.ndjson")
	if err := os.WriteFile(controlPath, []byte(`{"id":"cmd-1","action":"pause"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	entry, ok := handler.findEntry("control command")
	if !ok {
		t.Fatal("expected 'control command' log entry")
	}
	if entry.Attrs["action"] != "pause" {
		t.Fatalf("action = %v, want pause", entry.Attrs["action"])
	}
}

func TestLogControlCommandFailed(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	controlPath := filepath.Join(tc.runtimeDir, "control.ndjson")
	if err := os.WriteFile(controlPath, []byte(`{"id":"cmd-1","action":"kill","name":"nonexistent"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	entry, ok := handler.findEntry("control command failed")
	if !ok {
		t.Fatal("expected 'control command failed' log entry")
	}
	if entry.Level != slog.LevelWarn {
		t.Fatalf("expected Warn level, got %v", entry.Level)
	}
	if entry.Attrs["action"] != "kill" {
		t.Fatalf("action = %v, want kill", entry.Attrs["action"])
	}
	if entry.Attrs["message"] == nil || entry.Attrs["message"] == "" {
		t.Fatal("expected non-empty message attr")
	}
}

func TestLogBootstrapSchedule(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
		o.brief = &brief
	})

	// No orders file → empty orders → triggers bootstrap schedule.
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if !handler.hasMessage("orders empty, bootstrapping schedule") {
		t.Fatal("expected 'orders empty, bootstrapping schedule' log entry")
	}
}

func TestLogBootstrapScheduleDoesNotLoopWhenScheduleOrderExists(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
		o.brief = &brief
	})

	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     ScheduleTaskKey(),
				Title:  "scheduling tasks based on your backlog",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  ScheduleTaskKey(),
						Skill:    "schedule",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusPending,
					},
				},
			},
		},
	}
	if err := writeOrdersAtomic(tc.ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	if got := handler.countMessage("orders empty, bootstrapping schedule"); got != 0 {
		t.Fatalf("bootstrap log count = %d, want 0", got)
	}
}

func TestLogOrderRemoved(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	orders := OrdersFile{Orders: []Order{testOrder("item-1", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := tc.loop.writeOrdersState(orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	if err := tc.loop.removeOrder("item-1"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	entry, ok := handler.findEntry("order removed")
	if !ok {
		t.Fatal("expected 'order removed' log entry")
	}
	if entry.Attrs["order"] != "item-1" {
		t.Fatalf("order = %v, want item-1", entry.Attrs["order"])
	}
}

func TestLogIdleTransition(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	// No plans, no queue → should go idle.
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	entry, ok := handler.findEntry("state changed")
	if !ok {
		t.Fatal("expected 'state changed' log entry for idle transition")
	}
	if entry.Attrs["to"] != "idle" {
		t.Fatalf("to = %v, want idle", entry.Attrs["to"])
	}
}

func TestLogOrdersNextPromoted(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger, func(o *testLoopOpts) {
		brief := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
		o.brief = &brief
	})

	// Write a valid orders-next.json.
	ordersNextPath := filepath.Join(tc.runtimeDir, "orders-next.json")
	nextOrders := OrdersFile{Orders: []Order{{
		ID: "from-schedule", Title: "from schedule", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending}},
	}}}
	if err := writeOrdersAtomic(ordersNextPath, nextOrders); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}
	tc.loop.deps.OrdersNextFile = ordersNextPath

	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if !handler.hasMessage("orders-next promoted") {
		// Check if it was a failure instead
		if handler.hasMessage("orders-next promotion failed") {
			entry, _ := handler.findEntry("orders-next promotion failed")
			t.Fatalf("orders-next promotion failed: %v", entry.Attrs["error"])
		}
		t.Fatal("expected 'orders-next promoted' log entry")
	}
}

func TestLogResumeStateTransition(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	// Pause first.
	controlPath := filepath.Join(tc.runtimeDir, "control.ndjson")
	if err := os.WriteFile(controlPath, []byte(`{"id":"cmd-1","action":"pause"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("pause cycle: %v", err)
	}

	// Resume.
	if err := os.WriteFile(controlPath, []byte(`{"id":"cmd-2","action":"resume"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := tc.loop.Cycle(context.Background()); err != nil {
		t.Fatalf("resume cycle: %v", err)
	}

	// Should have two state changes: running→paused, paused→running.
	entries := handler.snapshot()
	stateChanges := 0
	var transitions []string
	for _, e := range entries {
		if e.Message == "state changed" {
			stateChanges++
			from, _ := e.Attrs["from"].(string)
			to, _ := e.Attrs["to"].(string)
			transitions = append(transitions, from+"→"+to)
		}
	}
	if stateChanges < 2 {
		t.Fatalf("expected at least 2 state changes, got %d: %v", stateChanges, transitions)
	}

	// Verify the pause→resume pair.
	foundPause := false
	foundResume := false
	for _, tr := range transitions {
		if strings.Contains(tr, "running→paused") {
			foundPause = true
		}
		if strings.Contains(tr, "paused→running") {
			foundResume = true
		}
	}
	if !foundPause {
		t.Fatalf("missing running→paused transition, got %v", transitions)
	}
	if !foundResume {
		t.Fatalf("missing paused→running transition, got %v", transitions)
	}
}

func TestLogRuntimeDispatchFallback(t *testing.T) {
	logger, handler := newTestLogger()
	tc := newTestLoop(t, logger)

	remoteRuntime := newMockRuntime()
	remoteRuntime.dispatchErr = errors.New("remote runtime unavailable")
	tc.loop.deps.Runtimes["sprites"] = remoteRuntime

	req := loopruntime.DispatchRequest{
		Name:    "fallback-test",
		Prompt:  "do something",
		Runtime: "sprites",
	}
	if _, _, err := tc.loop.dispatchSession(context.Background(), req); err != nil {
		t.Fatalf("dispatchSession: %v", err)
	}

	entry, ok := handler.findEntry("runtime dispatch failed, falling back to process")
	if !ok {
		t.Fatal("expected runtime fallback warning log")
	}
	if entry.Attrs["runtime"] != "sprites" {
		t.Fatalf("runtime attr = %v, want sprites", entry.Attrs["runtime"])
	}
}
