package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/skill"
)

func writeTaskTypeSkill(t *testing.T, projectDir, name, scheduleHint string) {
	t.Helper()
	skillDir := filepath.Join(projectDir, ".agents", "skills", name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir %s: %v", name, err)
	}
	content := fmt.Sprintf(`---
name: %s
description: %s
schedule: %q
---

# %s
`, name, name, scheduleHint, name)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill %s: %v", name, err)
	}
}

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

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	cfg.Concurrency.MaxConcurrency = orderCount
	cfg.Runtime.Process.MaxConcurrent = orderCount

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if len(rt.sessions) != orderCount {
		t.Fatalf("dispatched sessions = %d, want %d", len(rt.sessions), orderCount)
	}

	for _, session := range rt.sessions {
		session.complete("completed")
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

func TestScaleLoopStateSnapshotIncludesActiveSummary(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		testOrder("snap-1", "execute", "execute", "claude", "claude-opus-4-6"),
		testOrder("snap-2", "execute", "execute", "claude", "claude-opus-4-6"),
	}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	state := l.State()
	if state.Status != string(StateRunning) {
		t.Fatalf("state status = %q, want %q", state.Status, StateRunning)
	}
	if len(state.ActiveCooks) != 2 {
		t.Fatalf("active cooks = %d, want 2", len(state.ActiveCooks))
	}
	if state.ActiveSummary.Total != 2 {
		t.Fatalf("active summary total = %d, want 2", state.ActiveSummary.Total)
	}
}

// TestMockRuntimeDispatchAndComplete verifies the full dispatch→complete
// lifecycle using MockRuntime through the Runtimes map path.
func TestMockRuntimeDispatchAndComplete(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		testOrder("rt-1", "execute", "execute", "claude", "claude-opus-4-6"),
		testOrder("rt-2", "execute", "execute", "claude", "claude-opus-4-6"),
	}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
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
	if len(rt.sessions) != 2 {
		t.Fatalf("dispatched sessions = %d, want 2", len(rt.sessions))
	}

	// Complete all sessions.
	for _, s := range rt.sessions {
		s.complete("completed")
	}

	for i := 0; i < 10; i++ {
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

	current, _ := readOrders(ordersPath)
	t.Fatalf("orders remaining: %d", len(current.Orders))
}

// TestMockRuntimeRecoverBuildsAdoptedIndex verifies that Runtime.Recover()
// results are used to build the adopted session index during reconcile.
func TestMockRuntimeRecoverBuildsAdoptedIndex(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "order-1", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Provider: "claude", Model: "opus", Status: StageStatusActive}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.recovered = []loopruntime.RecoveredSession{
		{
			OrderID:       "order-1",
			SessionHandle: &mockSession{id: "sess-1", status: "running", done: make(chan struct{})},
			RuntimeName:   "process",
			Reason:        "test recovery",
		},
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if sid, ok := l.cooks.adoptedTargets["order-1"]; !ok {
		t.Fatal("expected order-1 in adoptedTargets")
	} else if sid != "sess-1" {
		t.Fatalf("adoptedTargets[order-1] = %q, want sess-1", sid)
	}
	if len(l.cooks.adoptedSessions) != 1 {
		t.Fatalf("adoptedSessions = %d, want 1", len(l.cooks.adoptedSessions))
	}
}

func TestRecoverAdoptedSessionsUpdatesActiveSummary(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "42", Status: OrderStatusActive, Stages: []Stage{
			{TaskKey: "execute", Runtime: "process", Provider: "claude", Model: "opus", Status: StageStatusActive},
		}},
		{ID: "99", Status: OrderStatusActive, Stages: []Stage{
			{TaskKey: "quality", Runtime: "sprites", Provider: "codex", Model: "gpt-5", Status: StageStatusActive},
		}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.recovered = []loopruntime.RecoveredSession{
		{
			OrderID:       "42",
			SessionHandle: &mockSession{id: "sess-42", status: "running", done: make(chan struct{})},
			RuntimeName:   "process",
		},
		{
			OrderID:       "99",
			SessionHandle: &mockSession{id: "sess-99", status: "running", done: make(chan struct{})},
			RuntimeName:   "sprites",
		},
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	snap := l.snapshotActiveSummary()
	if snap.Total != 2 {
		t.Fatalf("activeSummary.Total = %d, want 2", snap.Total)
	}
	if snap.ByTaskKey["execute"] != 1 {
		t.Fatalf("ByTaskKey[execute] = %d, want 1", snap.ByTaskKey["execute"])
	}
	if snap.ByTaskKey["quality"] != 1 {
		t.Fatalf("ByTaskKey[quality] = %d, want 1", snap.ByTaskKey["quality"])
	}
	if snap.ByRuntime["process"] != 1 {
		t.Fatalf("ByRuntime[process] = %d, want 1", snap.ByRuntime["process"])
	}
	if snap.ByRuntime["sprites"] != 1 {
		t.Fatalf("ByRuntime[sprites] = %d, want 1", snap.ByRuntime["sprites"])
	}
	if snap.ByStatus["active"] != 2 {
		t.Fatalf("ByStatus[active] = %d, want 2", snap.ByStatus["active"])
	}
}

func TestReconcileInjectsScheduleOrderOnStartup(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if !hasScheduleOrder(updated) {
		t.Fatalf("expected startup reconcile to inject schedule order, got %#v", updated.Orders)
	}
}

func TestReconcileInjectsOopsBootstrapOrderWhenScheduleMissing(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	writeTaskTypeSkill(t, projectDir, "oops", "When infrastructure failures are detected")
	writeTaskTypeSkill(t, projectDir, "execute", "When a planned item is ready")

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	cfg.Skills.Paths = []string{filepath.Join(projectDir, ".agents", "skills")}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{
		hasUnmergedCommits: map[string]bool{
			"oops-bootstrap-schedule-0-oops": false,
		},
	}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) == 0 || updated.Orders[0].ID != scheduleBootstrapOrderID {
		t.Fatalf("expected startup reconcile to inject %q, got %#v", scheduleBootstrapOrderID, updated.Orders)
	}
	stage := updated.Orders[0].Stages[0]
	if stage.TaskKey != "oops" || stage.Skill != "oops" {
		t.Fatalf("bootstrap order stage = %#v, want oops task+skill", stage)
	}
	if !strings.Contains(stage.Prompt, "github.com/poteto/noodle/.agents/skills/schedule/") {
		t.Fatalf("bootstrap prompt missing schedule skill example reference: %q", stage.Prompt)
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(rt.calls) != 1 || rt.calls[0].Skill != "oops" {
		t.Fatalf("expected immediate oops dispatch, calls=%#v", rt.calls)
	}
}

func TestOopsBootstrapCompletionInjectsScheduleOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	writeTaskTypeSkill(t, projectDir, "oops", "When infrastructure failures are detected")
	writeTaskTypeSkill(t, projectDir, "execute", "When a planned item is ready")

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	cfg.Skills.Paths = []string{filepath.Join(projectDir, ".agents", "skills")}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{
		hasUnmergedCommits: map[string]bool{
			"oops-bootstrap-schedule-0-oops": false,
		},
	}
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("dispatch oops cycle: %v", err)
	}
	if len(rt.sessions) != 1 {
		t.Fatalf("expected bootstrap oops session, got %d sessions", len(rt.sessions))
	}

	// Simulate the oops agent having created and merged a schedule skill.
	writeTaskTypeSkill(t, projectDir, "schedule", "When orders are empty")
	rt.sessions[0].complete("completed")
	time.Sleep(20 * time.Millisecond)

	for i := 0; i < 5; i++ {
		if err := l.Cycle(context.Background()); err != nil {
			t.Fatalf("completion cycle %d: %v", i+1, err)
		}
		if len(rt.calls) >= 2 && rt.calls[1].Skill == "schedule" {
			return
		}
	}
	t.Fatalf("expected schedule dispatch after bootstrap completion, calls=%#v", rt.calls)
}

func TestReconcileAndCycleDispatchBootstrapWhenNoScheduleOrOopsTaskType(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	reg := taskreg.NewFromSkills([]skill.SkillMeta{
		{
			Name:        "execute",
			Path:        "/skills/execute",
			Frontmatter: skill.Frontmatter{Schedule: "When a planned item is ready"},
		},
	})

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   reg,
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) == 0 || updated.Orders[0].ID != scheduleOrderID {
		t.Fatalf("expected fallback schedule order to remain, got %#v", updated.Orders)
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(rt.calls) != 1 {
		t.Fatalf("expected one bootstrap dispatch call, got %d (%#v)", len(rt.calls), rt.calls)
	}
	req := rt.calls[0]
	if req.Name != bootstrapSessionPrefix+scheduleOrderID {
		t.Fatalf("bootstrap dispatch name = %q, want %q", req.Name, bootstrapSessionPrefix+scheduleOrderID)
	}
	if strings.TrimSpace(req.Skill) != "" {
		t.Fatalf("bootstrap dispatch skill = %q, want empty", req.Skill)
	}
	if strings.TrimSpace(req.SystemPrompt) == "" {
		t.Fatal("bootstrap dispatch should carry system prompt")
	}
}

func TestReconcileResetsStaleActiveScheduleStageAndCycleDispatches(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     ScheduleTaskKey(),
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  ScheduleTaskKey(),
						Skill:    "schedule",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusActive,
					},
				},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{}, // empty backlog: startup should still dispatch schedule
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("loadOrdersState: %v", err)
	}
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 || updated.Orders[0].Stages[0].Status != StageStatusPending {
		t.Fatalf("expected stale active schedule stage reset to pending, got %#v", updated.Orders)
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(rt.calls) != 1 || rt.calls[0].Skill != "schedule" {
		t.Fatalf("expected schedule dispatch after startup reconcile, calls=%#v", rt.calls)
	}
}

func TestReconcilePreservesActiveScheduleStageWhenRecovered(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     ScheduleTaskKey(),
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  ScheduleTaskKey(),
						Skill:    "schedule",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusActive,
					},
				},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.recovered = []loopruntime.RecoveredSession{
		{
			OrderID:       ScheduleTaskKey(),
			SessionHandle: &mockSession{id: "sched-1", status: "running", done: make(chan struct{})},
			RuntimeName:   "process",
			Reason:        "test recovery",
		},
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 || updated.Orders[0].Stages[0].Status != StageStatusActive {
		t.Fatalf("expected recovered active schedule stage to remain active, got %#v", updated.Orders)
	}
	if sid, ok := l.cooks.adoptedTargets[ScheduleTaskKey()]; !ok || sid != "sched-1" {
		t.Fatalf("adopted schedule session = %q, present=%v", sid, ok)
	}
}

func TestReconcileResetsStaleActiveNonScheduleStageAndCycleDispatches(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	const staleOrderID = "fix-uncommitted-ui-changes"
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     staleOrderID,
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  "oops",
						Skill:    "oops",
						Provider: "claude",
						Model:    "claude-opus-4-6",
						Status:   StageStatusActive,
					},
				},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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
	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	foundPending := false
	for _, order := range updated.Orders {
		if order.ID == staleOrderID && len(order.Stages) == 1 && order.Stages[0].Status == StageStatusPending {
			foundPending = true
			break
		}
	}
	if !foundPending {
		t.Fatalf("expected stale active stage reset to pending for %q, got %#v", staleOrderID, updated.Orders)
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	dispatched := false
	for _, call := range rt.calls {
		if call.Name == "fix-uncommitted-ui-changes-0-oops" {
			dispatched = true
			break
		}
	}
	if !dispatched {
		t.Fatalf("expected stale order %q to dispatch after reconcile; calls=%#v", staleOrderID, rt.calls)
	}
}

// TestMockRuntimeScaleBurstViaRuntimes mirrors TestScaleBurstCompletionProcessesAllOrders
// but dispatches through the Runtimes map instead of the legacy Dispatcher fallback.
func TestMockRuntimeScaleBurstViaRuntimes(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	const orderCount = 50
	orders := OrdersFile{Orders: make([]Order, 0, orderCount)}
	for i := 0; i < orderCount; i++ {
		id := fmt.Sprintf("rt-scale-%03d", i)
		orders.Orders = append(orders.Orders, testOrder(id, "execute", "execute", "claude", "claude-opus-4-6"))
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	cfg.Concurrency.MaxConcurrency = orderCount

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
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
	if len(rt.sessions) != orderCount {
		t.Fatalf("dispatched sessions = %d, want %d", len(rt.sessions), orderCount)
	}

	for _, s := range rt.sessions {
		s.complete("completed")
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

	current, _ := readOrders(ordersPath)
	t.Fatalf("orders remaining after burst: %d", len(current.Orders))
}
