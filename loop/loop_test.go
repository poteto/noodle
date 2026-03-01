package loop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/failure"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/worktree"
)

type fakeWorktree struct {
	created         []string
	merged          []string
	remoteMerged    []string
	cleaned         []string
	mergeErr        error
	remoteMergeErr  error
	createErr       error
	createErrByName map[string]error
	hasUnmergedCommits map[string]bool
}

func (f *fakeWorktree) Create(name string, _ ...worktree.CreateOpts) error {
	f.created = append(f.created, name)
	if f.createErrByName != nil {
		if err, ok := f.createErrByName[name]; ok {
			return err
		}
	}
	return f.createErr
}
func (f *fakeWorktree) Merge(name string, _ ...worktree.MergeOpts) error {
	f.merged = append(f.merged, name)
	return f.mergeErr
}
func (f *fakeWorktree) MergeRemoteBranch(branch string) error {
	f.remoteMerged = append(f.remoteMerged, branch)
	return f.remoteMergeErr
}
func (f *fakeWorktree) Cleanup(name string, _ ...worktree.CleanupOpts) error {
	f.cleaned = append(f.cleaned, name)
	return nil
}
func (f *fakeWorktree) HasUnmergedCommits(name string) (bool, error) {
	if f.hasUnmergedCommits != nil {
		if v, ok := f.hasUnmergedCommits[name]; ok {
			return v, nil
		}
	}
	return true, nil
}

type fakeAdapterRunner struct {
	doneCalls []string
}

func (f *fakeAdapterRunner) Run(_ context.Context, adapterName, action string, opts adapter.RunOptions) (string, error) {
	if adapterName == "backlog" && action == "done" && len(opts.Args) > 0 {
		f.doneCalls = append(f.doneCalls, opts.Args[0])
	}
	return "", nil
}

type fakeMise struct {
	brief    mise.Brief
	warnings []string
	err      error
	results  []fakeMiseResult
	calls    int
}

func (f *fakeMise) Build(_ context.Context, _ mise.ActiveSummary, _ []mise.HistoryItem) (mise.Brief, []string, bool, error) {
	f.calls++
	if len(f.results) > 0 {
		index := f.calls - 1
		if index >= len(f.results) {
			index = len(f.results) - 1
		}
		current := f.results[index]
		return current.brief, current.warnings, false, current.err
	}
	return f.brief, f.warnings, false, f.err
}

type fakeMiseResult struct {
	brief    mise.Brief
	warnings []string
	err      error
}

type fakeMonitor struct{}

func (fakeMonitor) RunOnce(_ context.Context) ([]monitor.SessionMeta, error) {
	return nil, nil
}

// writeTestOrders writes orders.json from an OrdersFile.
func writeTestOrders(t *testing.T, runtimeDir string, orders OrdersFile) string {
	t.Helper()
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	return ordersPath
}

// testOrder creates a single-stage active Order for test convenience.
func testOrder(id, taskKey, skill, provider, model string) Order {
	return Order{
		ID:     id,
		Status: OrderStatusActive,
		Stages: []Stage{{
			TaskKey:  taskKey,
			Skill:    skill,
			Provider: provider,
			Model:    model,
			Status:   StageStatusPending,
		}},
	}
}

func processSingleControlCommand(t *testing.T, commandJSON string) ControlAck {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	control := filepath.Join(runtimeDir, "control.ndjson")
	if err := os.WriteFile(control, []byte(commandJSON+"\n"), 0o644); err != nil {
		t.Fatalf("write control command: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      func() time.Time { return time.Date(2026, 2, 22, 23, 0, 0, 0, time.UTC) },
	})
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("process commands: %v", err)
	}

	ackPath := filepath.Join(runtimeDir, "control-ack.ndjson")
	data, err := os.ReadFile(ackPath)
	if err != nil {
		t.Fatalf("read ack file: %v", err)
	}
	var ack ControlAck
	if err := json.Unmarshal(data[:len(data)-1], &ack); err != nil {
		t.Fatalf("parse ack: %v", err)
	}
	return ack
}

func assertFailureStateMessage(t *testing.T, message string) {
	t.Helper()
	isWordBoundary := func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
	}
	words := strings.FieldsFunc(strings.ToLower(message), isWordBoundary)
	for _, term := range []string{"must", "required", "requires", "expected"} {
		for _, word := range words {
			if word == term {
				t.Fatalf("message %q contains expectation-style term %q", message, term)
			}
		}
	}
}

func TestCycleSpawnsCookFromOrders(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := writeTestOrders(t, runtimeDir, orders)

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(rt.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(rt.calls))
	}
	if len(wt.created) != 1 {
		t.Fatalf("worktree creates = %d", len(wt.created))
	}
}

func TestCycleReusesExistingWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42-0-execute")
	if err := os.MkdirAll(existingWorktree, 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(wt.created) != 0 {
		t.Fatalf("expected no worktree create calls, got %d", len(wt.created))
	}
	if len(rt.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(rt.calls))
	}
	if rt.calls[0].WorktreePath != existingWorktree {
		t.Fatalf("spawn worktree path = %q, want %q", rt.calls[0].WorktreePath, existingWorktree)
	}
}

func TestCycleIgnoresDuplicateWorktreeCreateError(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	existingWorktree := filepath.Join(projectDir, ".worktrees", "42")
	rt := newMockRuntime()
	wt := &fakeWorktree{createErr: errors.New("worktree '42' already exists at " + existingWorktree)}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(wt.created) != 1 {
		t.Fatalf("expected one create call, got %d", len(wt.created))
	}
	if len(rt.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(rt.calls))
	}
}

func TestCycleSpawnFailureDoesNotCleanupReusedWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, ".worktrees", "42-0-execute"), 0o755); err != nil {
		t.Fatalf("mkdir existing worktree: %v", err)
	}

	rt := newMockRuntime()
	rt.dispatchErr = errors.New("spawn failed")
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	for _, name := range wt.cleaned {
		if name == "42-0-execute" {
			t.Fatalf("expected no cleanup for reused worktree, got %#v", wt.cleaned)
		}
	}
	got, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].Status != OrderStatusFailed {
		t.Fatalf("order status = %q, want %q", got.Orders[0].Status, OrderStatusFailed)
	}
	if status := got.Orders[0].Stages[0].Status; status != StageStatusFailed {
		t.Fatalf("stage status = %q, want %q", status, StageStatusFailed)
	}
}

func TestCycleSpawnFailureCleansUpNewWorktree(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.dispatchErr = errors.New("spawn failed")
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	created42 := false
	for _, name := range wt.created {
		if name == "42-0-execute" {
			created42 = true
			break
		}
	}
	if !created42 {
		t.Fatalf("expected create call for new worktree, got %#v", wt.created)
	}
	cleaned42 := false
	for _, name := range wt.cleaned {
		if name == "42-0-execute" {
			cleaned42 = true
			break
		}
	}
	if !cleaned42 {
		t.Fatalf("expected cleanup for newly created worktree, got %#v", wt.cleaned)
	}
	got, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].Status != OrderStatusFailed {
		t.Fatalf("order status = %q, want %q", got.Orders[0].Status, OrderStatusFailed)
	}
	if status := got.Orders[0].Stages[0].Status; status != StageStatusFailed {
		t.Fatalf("stage status = %q, want %q", status, StageStatusFailed)
	}
}

func TestCycleRetryableDispatchFailureResetsStagePending(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "42",
				Status: OrderStatusActive,
				Stages: []Stage{{
					TaskKey:  "execute",
					Skill:    "execute",
					Provider: "claude",
					Model:    "claude-opus-4-6",
					Runtime:  "sprites",
					Status:   StageStatusPending,
				}},
			},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	spritesRT := newMockRuntime()
	spritesRT.dispatchErr = errors.New("sprites runtime unavailable")
	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"sprites": spritesRT},
		Worktree:   wt,
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
	got, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", got.Orders[0].Status, OrderStatusActive)
	}
	if status := got.Orders[0].Stages[0].Status; status != StageStatusPending {
		t.Fatalf("stage status = %q, want %q", status, StageStatusPending)
	}
}

func TestCycleCompletesCookAndMarksDone(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  ar,
		Mise:     &fakeMise{brief: briefWithPlans},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(rt.sessions) != 1 {
		t.Fatalf("sessions = %d", len(rt.sessions))
	}
	rt.sessions[0].complete("completed")

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	if len(wt.merged) != 1 {
		t.Fatalf("worktree merges = %d", len(wt.merged))
	}
	if len(ar.doneCalls) != 1 || ar.doneCalls[0] != "42" {
		t.Fatalf("backlog done calls = %#v", ar.doneCalls)
	}
	// Verify the order was removed from orders.json after completion.
	updatedOrders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders after completion: %v", err)
	}
	for _, order := range updatedOrders.Orders {
		if order.ID == "42" {
			t.Fatal("order 42 should have been removed after completion")
		}
	}
}

func TestCycleMergeFailureForwardsToSchedulerWithoutCrashing(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := writeTestOrders(t, runtimeDir, orders)

	rt := newMockRuntime()
	wt := &fakeWorktree{
		mergeErr: errors.New("merge 20-0-execute: root checkout has uncommitted changes — commit or stash before merging"),
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	ctrl := &mockController{steerable: true}
	l.cooks.activeCooksByOrder[ScheduleTaskKey()] = &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    ScheduleTaskKey(),
			stageIndex: 0,
			stage: Stage{
				TaskKey: ScheduleTaskKey(),
				Skill:   "schedule",
				Status:  StageStatusActive,
			},
		},
		session: &steerableSession{
			mockSession: &mockSession{id: "scheduler-session", status: "running", done: make(chan struct{})},
			ctrl:        ctrl,
		},
	}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(rt.sessions) != 1 {
		t.Fatalf("sessions = %d", len(rt.sessions))
	}
	rt.sessions[0].complete("completed")

	// Merge failure should be handled as a stage failure and forwarded to the
	// scheduler, not returned as a fatal loop cycle error.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle should not crash on merge failure: %v", err)
	}

	updatedOrders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders after merge failure: %v", err)
	}
	found := false
	for _, order := range updatedOrders.Orders {
		if order.ID == "42" {
			found = true
			if order.Status != OrderStatusFailed {
				t.Fatalf("order status = %q, want %q", order.Status, OrderStatusFailed)
			}
			if got := order.Stages[0].Status; got != StageStatusFailed {
				t.Fatalf("stage status = %q, want %q", got, StageStatusFailed)
			}
		}
	}
	if !found {
		t.Fatal("order 42 should remain in orders.json after merge failure")
	}
	if _, ok := l.cooks.pendingReview["42"]; ok {
		t.Fatal("non-conflict merge failure should not park pending review")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ctrl.mu.Lock()
		sent := ctrl.sendCalls
		ctrl.mu.Unlock()
		if sent > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	if ctrl.sendCalls == 0 {
		t.Fatal("expected merge failure message to be forwarded to scheduler")
	}
	if !strings.Contains(ctrl.lastSentMessage, "[merge_failed]") {
		t.Fatalf("scheduler message = %q, want merge_failed tag", ctrl.lastSentMessage)
	}
	if !strings.Contains(ctrl.lastSentMessage, "root checkout has uncommitted changes") {
		t.Fatalf("scheduler message = %q, want root-cleanliness error", ctrl.lastSentMessage)
	}
	if !strings.Contains(ctrl.lastSentMessage, "Use Skill(schedule) and create a new order to fix this issue before retrying.") {
		t.Fatalf("scheduler message = %q, want Skill(schedule) remediation hint", ctrl.lastSentMessage)
	}
}

func TestCycleEntersIdleWhenNoPlansRemain(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if l.state != StateIdle {
		t.Fatalf("state = %s, want idle", l.state)
	}
	if len(rt.calls) != 0 {
		t.Fatalf("expected no spawn calls when idle, got %d", len(rt.calls))
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.LoopState != "idle" {
		t.Fatalf("status loop_state = %q, want idle", status.LoopState)
	}
}

func TestCycleIdleWakesWhenPlansAppear(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	fm := &fakeMise{}
	rt := newMockRuntime()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     fm,
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	// First cycle: no plans → idle
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if l.state != StateIdle {
		t.Fatalf("state after cycle 1 = %s, want idle", l.state)
	}

	// Simulate new backlog items appearing
	fm.brief = mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}

	// Second cycle: idle → running, bootstraps schedule
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if l.state != StateRunning {
		t.Fatalf("state after cycle 2 = %s, want running", l.state)
	}
	if len(rt.calls) != 1 {
		t.Fatalf("expected 1 spawn call after wake, got %d", len(rt.calls))
	}
	if rt.calls[0].Skill != "schedule" {
		t.Fatalf("expected schedule spawn, got skill %q", rt.calls[0].Skill)
	}
}

func TestCycleBootstrapsScheduleUsesRegistrySkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	if len(rt.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(rt.calls))
	}
	if rt.calls[0].Skill != "schedule" {
		t.Fatalf("spawn skill = %q", rt.calls[0].Skill)
	}
	expectedMise := filepath.Join(runtimeDir, "mise.json")
	if !strings.Contains(rt.calls[0].Prompt, "Use Skill(schedule) to refresh the schedule from "+expectedMise+".") {
		t.Fatalf("spawn prompt missing skill invocation: %q", rt.calls[0].Prompt)
	}
	expectedOrdersNext := filepath.Join(runtimeDir, "orders-next.json")
	if !strings.Contains(rt.calls[0].Prompt, expectedOrdersNext) {
		t.Fatalf("spawn prompt missing orders-next.json instruction: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(rt.calls[0].Prompt, "orders.json schema (JSON):") {
		t.Fatalf("spawn prompt missing orders schema: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(rt.calls[0].Prompt, "Task types you may schedule:") {
		t.Fatalf("spawn prompt missing task type catalog: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(rt.calls[0].Prompt, "- schedule: ") || !strings.Contains(rt.calls[0].Prompt, "- execute: ") {
		t.Fatalf("spawn prompt missing key+schedule task type guidance: %q", rt.calls[0].Prompt)
	}
	if strings.Contains(rt.calls[0].Prompt, "| config: ") || strings.Contains(rt.calls[0].Prompt, "| synthetic: ") {
		t.Fatalf("spawn prompt should not include verbose task type metadata: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(rt.calls[0].Prompt, "Do not modify "+expectedMise+".") {
		t.Fatalf("spawn prompt missing mise immutability note: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(rt.calls[0].Prompt, "Operate fully autonomously. Never ask the user questions.") {
		t.Fatalf("spawn prompt missing autonomous mode note: %q", rt.calls[0].Prompt)
	}
	if !strings.Contains(
		rt.calls[0].Prompt,
		"You may synthesize orders for non-execute task types",
	) {
		t.Fatalf("spawn prompt missing synthesized-order guidance: %q", rt.calls[0].Prompt)
	}
	if strings.Contains(rt.calls[0].Prompt, "mise.json schema (JSON):") {
		t.Fatalf("spawn prompt must not include mise schema: %q", rt.calls[0].Prompt)
	}
	if !rt.calls[0].AllowPrimaryCheckout {
		t.Fatal("expected schedule spawn to allow primary checkout")
	}
	if rt.calls[0].WorktreePath != projectDir {
		t.Fatalf("worktree path = %q, want %q", rt.calls[0].WorktreePath, projectDir)
	}
	if len(wt.created) != 0 {
		t.Fatalf("unexpected worktree creates: %#v", wt.created)
	}

	// Verify the schedule order was bootstrapped into orders.json.
	updatedOrders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updatedOrders.Orders) != 1 || updatedOrders.Orders[0].ID != ScheduleTaskKey() {
		t.Fatalf("expected schedule bootstrap order, got %#v", updatedOrders.Orders)
	}
	if len(updatedOrders.Orders[0].Stages) < 1 || updatedOrders.Orders[0].Stages[0].Skill != "schedule" {
		t.Fatalf("schedule order skill mismatch: %#v", updatedOrders.Orders[0].Stages)
	}
}

func TestProcessControlCommandsPauseAndAck(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	control := filepath.Join(runtimeDir, "control.ndjson")
	if err := os.WriteFile(control, []byte(`{"id":"cmd-1","action":"pause"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      func() time.Time { return time.Date(2026, 2, 22, 23, 0, 0, 0, time.UTC) },
	})
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("process commands: %v", err)
	}
	if l.state != StatePaused {
		t.Fatalf("state = %s", l.state)
	}

	ackPath := filepath.Join(runtimeDir, "control-ack.ndjson")
	data, err := os.ReadFile(ackPath)
	if err != nil {
		t.Fatalf("read ack file: %v", err)
	}
	var ack ControlAck
	if err := json.Unmarshal(data[:len(data)-1], &ack); err != nil {
		t.Fatalf("parse ack: %v", err)
	}
	if ack.ID != "cmd-1" || ack.Status != "ok" {
		t.Fatalf("ack = %#v", ack)
	}
	if ack.Failure != nil {
		t.Fatalf("ack failure = %#v, want nil", ack.Failure)
	}
}

func TestProcessControlCommandsInvalidJSONWritesTypedFailureAck(t *testing.T) {
	ack := processSingleControlCommand(t, `{"id":"cmd-1","action":"pause"`)
	if ack.Status != "error" {
		t.Fatalf("status = %q, want error", ack.Status)
	}
	if ack.Message != "invalid command JSON" {
		t.Fatalf("message = %q, want invalid command JSON", ack.Message)
	}
	if ack.Failure == nil {
		t.Fatal("failure metadata missing from invalid-json ack")
	}
	if ack.Failure.Class != failure.FailureClassBackendRecoverable {
		t.Fatalf("failure class = %q, want %q", ack.Failure.Class, failure.FailureClassBackendRecoverable)
	}
	if ack.Failure.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("recoverability = %q, want %q", ack.Failure.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
	if ack.Failure.Owner != failure.FailureOwnerBackend {
		t.Fatalf("owner = %q, want %q", ack.Failure.Owner, failure.FailureOwnerBackend)
	}
	if ack.Failure.Scope != failure.FailureScopeSystem {
		t.Fatalf("scope = %q, want %q", ack.Failure.Scope, failure.FailureScopeSystem)
	}
}

func TestProcessControlCommandsMissingStopTargetUsesFailureStateMessage(t *testing.T) {
	ack := processSingleControlCommand(t, `{"id":"cmd-1","action":"stop"}`)
	if ack.Status != "error" {
		t.Fatalf("status = %q, want error", ack.Status)
	}
	if ack.Message != "stop target missing" {
		t.Fatalf("message = %q, want %q", ack.Message, "stop target missing")
	}
	assertFailureStateMessage(t, ack.Message)
	if ack.Failure == nil {
		t.Fatal("failure metadata missing from stop ack")
	}
	if ack.Failure.Class != failure.FailureClassBackendRecoverable {
		t.Fatalf("failure class = %q, want %q", ack.Failure.Class, failure.FailureClassBackendRecoverable)
	}
	if ack.Failure.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("recoverability = %q, want %q", ack.Failure.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
	if ack.Failure.Owner != failure.FailureOwnerBackend {
		t.Fatalf("owner = %q, want %q", ack.Failure.Owner, failure.FailureOwnerBackend)
	}
	if ack.Failure.Scope != failure.FailureScopeSystem {
		t.Fatalf("scope = %q, want %q", ack.Failure.Scope, failure.FailureScopeSystem)
	}
}

func TestRemoveOrderMissingIDUsesFailureStateMessage(t *testing.T) {
	projectDir := t.TempDir()
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	err := l.removeOrder("")
	if err == nil {
		t.Fatal("expected removeOrder to fail for empty ID")
	}
	if err.Error() != "remove order ID missing" {
		t.Fatalf("message = %q, want %q", err.Error(), "remove order ID missing")
	}
	assertFailureStateMessage(t, err.Error())
}

func TestSteerScheduleRegeneratesOrdersWithPromptRationale(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise: &fakeMise{brief: mise.Brief{
			Backlog: []adapter.BacklogItem{{ID: "1", Title: "Fix", Status: adapter.BacklogStatusOpen}},
		}},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.steer(ScheduleTaskKey(), "schedule security tasks"); err != nil {
		t.Fatalf("steer schedule: %v", err)
	}
	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d", len(orders.Orders))
	}
	if orders.Orders[0].ID != ScheduleTaskKey() {
		t.Fatalf("unexpected id: %q", orders.Orders[0].ID)
	}
	if orders.Orders[0].Title == "Fix" {
		t.Fatalf("expected schedule order, got backlog item title %q", orders.Orders[0].Title)
	}
	if orders.Orders[0].Rationale != "Chef steer: schedule security tasks" {
		t.Fatalf("unexpected rationale: %q", orders.Orders[0].Rationale)
	}
}

func TestCookBaseNameIncludesOrderStageTask(t *testing.T) {
	name := cookBaseName("42", 0, "execute")
	if name != "42-0-execute" {
		t.Fatalf("unexpected cook name: %q", name)
	}
}

func TestCookBaseName(t *testing.T) {
	name := cookBaseName("42", 0, "execute")
	if name != "42-0-execute" {
		t.Fatalf("cookBaseName = %q, want 42-0-execute", name)
	}
}

func TestCookBaseNameDasherizesUnsafeTokens(t *testing.T) {
	name := cookBaseName("Plan 49/Phase:10", 2, "Request Changes")
	if name != "plan-49-phase-10-2-request-changes" {
		t.Fatalf("cookBaseName = %q, want plan-49-phase-10-2-request-changes", name)
	}
}

func TestCycleRegistryErrorBlocksAfterThreeFailures(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	// Create a loop with a registry error (simulates discovery failure)
	l := &Loop{
		projectDir:  projectDir,
		runtimeDir:  runtimeDir,
		registryErr: errors.New("task type discovery failed: bad frontmatter"),
		cooks: cookTracker{
			activeCooksByOrder: map[string]*cookHandle{},
			adoptedTargets:     map[string]string{},
			pendingReview:      map[string]*pendingReviewCook{},
		},
		cmds: cmdProcessor{
			processedIDs: map[string]struct{}{},
		},
		deps: Dependencies{
			Mise:       &fakeMise{brief: mise.Brief{}},
			Monitor:    fakeMonitor{},
			Now:        time.Now,
			StatusFile: filepath.Join(runtimeDir, "status.json"),
		},
	}

	// First two cycles skip with no error (resilient path).
	for i := 0; i < 2; i++ {
		err := l.Cycle(context.Background())
		if err != nil {
			t.Fatalf("cycle %d: expected no error, got: %v", i+1, err)
		}
	}

	// Third cycle returns the fatal registry error.
	err := l.Cycle(context.Background())
	if err == nil {
		t.Fatal("expected Cycle to return registry error after 3 failures")
	}
	if !strings.Contains(err.Error(), "task type discovery failed") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestReadSessionTargetAcceptsRichIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	content := "[order:plan/phase_02-ticket.7]\n\nContext: test"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	target := loopruntime.ReadSessionTarget(path)
	if target != "plan/phase_02-ticket.7" {
		t.Fatalf("target = %q", target)
	}
}

func TestReadSessionTargetDetectsSchedulePrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	content := "Use Skill(schedule) to refresh the queue from .noodle/mise.json."
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	target := loopruntime.ReadSessionTarget(path)
	if target != ScheduleTaskKey() {
		t.Fatalf("target = %q", target)
	}
}

func TestCycleRemovesStaleAdoptedSlotsBeforeScheduling(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", "stale-session"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", "stale-session", "meta.json"),
		[]byte(`{"status":"exited"}`),
		0o644,
	); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 1
	rt := newMockRuntime()
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	l.cooks.adoptedTargets = map[string]string{"legacy-1": "stale-session"}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(rt.calls) != 1 {
		t.Fatalf("spawn calls = %d", len(rt.calls))
	}
	if len(l.cooks.adoptedTargets) != 0 {
		t.Fatalf("expected stale adopted target to be removed, got %#v", l.cooks.adoptedTargets)
	}
}

func TestCycleStampsLoopStateWhenPaused(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Mode = "supervised"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	l.state = StatePaused

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Mode != "supervised" {
		t.Fatalf("mode = %q, want supervised", status.Mode)
	}
}

func TestCycleStampsLoopStateWhenDraining(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	l.state = StateDraining

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}

	statusPath := filepath.Join(runtimeDir, "status.json")
	status, err := statusfile.Read(statusPath)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Mode != "auto" {
		t.Fatalf("mode = %q, want auto", status.Mode)
	}
}

func TestCycleCompletesAdoptedCookFromMetaState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	sessionID := "adopted-session"
	if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
		t.Fatalf("mkdir session: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "meta.json"),
		[]byte(`{"status":"completed"}`),
		0o644,
	); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "prompt.txt"),
		[]byte("Work backlog item 42\n"),
		0o644,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	worktreePath := filepath.Join(projectDir, ".worktrees", "42")
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json"),
		[]byte(`{"worktree_path":"`+worktreePath+`"}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "42", Title: "test", Status: "open"}}}
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: wt,
		Adapter:  ar,
		Mise:     &fakeMise{brief: briefWithPlans},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})
	l.cooks.adoptedTargets = map[string]string{"42": sessionID}
	l.cooks.adoptedSessions = []string{sessionID}

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if len(wt.merged) != 1 || wt.merged[0] != "42-0-execute" {
		t.Fatalf("unexpected merged worktrees: %#v", wt.merged)
	}
	if len(ar.doneCalls) != 1 || ar.doneCalls[0] != "42" {
		t.Fatalf("unexpected done calls: %#v", ar.doneCalls)
	}
	if len(l.cooks.adoptedTargets) != 0 {
		t.Fatalf("expected adopted targets to be cleared, got %#v", l.cooks.adoptedTargets)
	}
	// Verify the order was removed from orders.json after completion.
	updatedOrders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders after adopted completion: %v", err)
	}
	for _, order := range updatedOrders.Orders {
		if order.ID == "42" {
			t.Fatal("order 42 should have been removed after completion")
		}
	}
}
