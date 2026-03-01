package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/worktree"
)

func TestMergeCookWorktreeUsesRemoteBranchSyncResult(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	sessionID := "session-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sessionPath, "spawn.json"),
		[]byte(`{"sync":{"type":"branch","branch":"noodle/session-a"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "42", stage: Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-sonnet-4-6"}},
		session:      &mockSession{id: sessionID, status: "completed", done: make(chan struct{})},
		worktreeName: "42",
	}
	if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
		t.Fatalf("mergeCookWorktree: %v", err)
	}
	if len(wt.remoteMerged) != 1 || wt.remoteMerged[0] != "noodle/session-a" {
		t.Fatalf("unexpected remote merges: %#v", wt.remoteMerged)
	}
	if len(wt.merged) != 0 {
		t.Fatalf("expected no local merge, got %#v", wt.merged)
	}
}

func TestMergeCookWorktreeFallsBackToLocalMerge(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	wt := &fakeWorktree{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "42", stage: Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-sonnet-4-6"}},
		session:      &mockSession{id: "", status: "completed", done: make(chan struct{})},
		worktreeName: "42",
	}
	if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
		t.Fatalf("mergeCookWorktree: %v", err)
	}
	if len(wt.merged) != 1 || wt.merged[0] != "42" {
		t.Fatalf("unexpected local merges: %#v", wt.merged)
	}
	if len(wt.remoteMerged) != 0 {
		t.Fatalf("expected no remote merge, got %#v", wt.remoteMerged)
	}
}

func TestCycleMergeConflictMarksFailedAndSkips(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-sonnet-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "42", Title: "test", Status: "open"}}}
	rt := newMockRuntime()
	wt := &fakeWorktree{
		remoteMergeErr: &worktree.MergeConflictError{Branch: "origin/noodle/session-a"},
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": rt},
		Worktree: wt,
		Adapter:  &fakeAdapterRunner{},
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

	sessionID := rt.sessions[0].id
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sessionPath, "spawn.json"),
		[]byte(`{"sync":{"type":"branch","branch":"noodle/session-a"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	rt.sessions[0].complete("completed")

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	// Merge conflicts now park for pending review instead of marking failed.
	if _, ok := l.cooks.pendingReview["42"]; !ok {
		t.Fatal("expected target to be parked for pending review after merge conflict")
	}
	pending := l.cooks.pendingReview["42"]
	if !strings.Contains(pending.reason, "merge conflict") {
		t.Fatalf("pending review reason = %q, want 'merge conflict'", pending.reason)
	}
}

func TestApprovalAutoCanMergeTrueAutoMerges(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// execute task type has CanMerge=true (no Merge override in registry)
	orders := OrdersFile{Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	ar := &fakeAdapterRunner{}
	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
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
		t.Fatalf("worktree merges = %d, want 1 (auto-merge)", len(wt.merged))
	}
	if len(l.cooks.pendingReview) != 0 {
		t.Fatalf("pendingReview should be empty, got %d item(s)", len(l.cooks.pendingReview))
	}
}

func TestApprovalAutoCanMergeFalseAdvances(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// review task type has CanMerge=false — in auto mode, non-mergeable stages advance without merge.
	orders := OrdersFile{Orders: []Order{testOrder("42", "review", "review", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}
	l := New(projectDir, "noodle", cfg, Dependencies{
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
		t.Fatalf("spawn cycle: %v", err)
	}
	if len(rt.sessions) != 1 {
		t.Fatalf("sessions = %d", len(rt.sessions))
	}
	rt.sessions[0].complete("completed")

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("completion cycle: %v", err)
	}
	if len(wt.merged) != 0 {
		t.Fatalf("worktree merges = %d, want 0 (task disallows merge)", len(wt.merged))
	}
	// In auto mode, non-mergeable stages advance without parking.
	if len(l.cooks.pendingReview) != 0 {
		t.Fatalf("pendingReview = %d, want 0 (auto mode advances non-mergeable stages)", len(l.cooks.pendingReview))
	}
	// Verify the order was removed from orders.json after completion.
	updatedOrders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	for _, order := range updatedOrders.Orders {
		if order.ID == "42" {
			t.Fatal("order 42 should have been removed after non-mergeable stage completion")
		}
	}
}

func TestPersistMergeMetadataWritesExtraFields(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{Orders: []Order{testOrder("order-1", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-1", stageIndex: 0},
		worktreeName: "order-1-0-execute",
		generation:   42,
		session:      &adoptedSession{id: "sess-1", status: "completed"},
	}

	if err := l.persistMergeMetadata(cook, "local", ""); err != nil {
		t.Fatalf("persistMergeMetadata: %v", err)
	}

	got, err := l.currentOrders()
	if err != nil {
		t.Fatalf("currentOrders: %v", err)
	}
	stage := got.Orders[0].Stages[0]
	if stage.Status != StageStatusMerging {
		t.Errorf("status = %q, want %q", stage.Status, StageStatusMerging)
	}
	assertExtra := func(key, want string) {
		t.Helper()
		var val string
		if err := json.Unmarshal(stage.Extra[key], &val); err != nil {
			t.Errorf("Extra[%s] unmarshal: %v", key, err)
			return
		}
		if val != want {
			t.Errorf("Extra[%s] = %q, want %q", key, val, want)
		}
	}
	assertExtra(mergeExtraWorktree, "order-1-0-execute")
	assertExtra(mergeExtraMode, "local")
	assertExtra(mergeExtraGeneration, "42")
	assertExtra(mergeExtraBranch, "")
}

func TestPersistMergeMetadataRemoteMode(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{Orders: []Order{testOrder("order-r", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-r", stageIndex: 0},
		worktreeName: "order-r-0-execute",
		generation:   7,
		session:      &adoptedSession{id: "sess-r", status: "completed"},
	}

	if err := l.persistMergeMetadata(cook, "remote", "noodle/remote-branch"); err != nil {
		t.Fatalf("persistMergeMetadata: %v", err)
	}

	got, err := l.currentOrders()
	if err != nil {
		t.Fatalf("currentOrders: %v", err)
	}
	stage := got.Orders[0].Stages[0]
	var mode string
	if err := json.Unmarshal(stage.Extra[mergeExtraMode], &mode); err != nil {
		t.Fatalf("unmarshal mode: %v", err)
	}
	if mode != "remote" {
		t.Errorf("mode = %q, want %q", mode, "remote")
	}
	var branch string
	if err := json.Unmarshal(stage.Extra[mergeExtraBranch], &branch); err != nil {
		t.Fatalf("unmarshal branch: %v", err)
	}
	if branch != "noodle/remote-branch" {
		t.Errorf("branch = %q, want %q", branch, "noodle/remote-branch")
	}
}

func TestReconcileMergingStagesMissingMetadataFails(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Stage is "merging" but has no Extra metadata.
	orders := OrdersFile{Orders: []Order{{
		ID:     "stuck-1",
		Status: OrderStatusActive,
		Stages: []Stage{{
			TaskKey: "execute",
			Skill:   "execute",
			Status:  StageStatusMerging,
		}},
	}}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.reconcileMergingStages(); err != nil {
		t.Fatalf("reconcileMergingStages: %v", err)
	}

	got, err := l.currentOrders()
	if err != nil {
		t.Fatalf("currentOrders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(got.Orders))
	}
	if got.Orders[0].Status != OrderStatusFailed {
		t.Errorf("order status = %q, want %q", got.Orders[0].Status, OrderStatusFailed)
	}
	if gotStage := got.Orders[0].Stages[0].Status; gotStage != StageStatusFailed {
		t.Errorf("stage status = %q, want %q", gotStage, StageStatusFailed)
	}
}

func TestReconcileMergingStagesAdoptedSessionResetsToActive(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{Orders: []Order{{
		ID:     "adopted-1",
		Status: OrderStatusActive,
		Stages: []Stage{{
			TaskKey: "execute",
			Skill:   "execute",
			Status:  StageStatusMerging,
			Extra: map[string]json.RawMessage{
				mergeExtraWorktree: jsonQuote("adopted-1-0-execute"),
				mergeExtraMode:     jsonQuote("local"),
			},
		}},
	}}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Simulate adopted session for this order.
	l.cooks.adoptedTargets = map[string]string{"adopted-1": "sess-adopted"}

	if err := l.reconcileMergingStages(); err != nil {
		t.Fatalf("reconcileMergingStages: %v", err)
	}

	got, err := l.currentOrders()
	if err != nil {
		t.Fatalf("currentOrders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(got.Orders))
	}
	if got.Orders[0].Stages[0].Status != StageStatusActive {
		t.Errorf("status = %q, want %q", got.Orders[0].Stages[0].Status, StageStatusActive)
	}
}

func TestReconcileMergingStagesNoMergingStagesIsNoop(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	orders := OrdersFile{Orders: []Order{testOrder("ok-1", "execute", "execute", "claude", "claude-opus-4-6")}}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.reconcileMergingStages(); err != nil {
		t.Fatalf("reconcileMergingStages: %v", err)
	}

	got, err := l.currentOrders()
	if err != nil {
		t.Fatalf("currentOrders: %v", err)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("expected 1 order unchanged, got %d", len(got.Orders))
	}
	if got.Orders[0].Stages[0].Status != StageStatusPending {
		t.Errorf("status = %q, want %q", got.Orders[0].Stages[0].Status, StageStatusPending)
	}
}

func TestExtraString(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]json.RawMessage
		key   string
		want  string
	}{
		{"nil map", nil, "key", ""},
		{"missing key", map[string]json.RawMessage{"other": jsonQuote("val")}, "key", ""},
		{"present", map[string]json.RawMessage{"key": jsonQuote("hello")}, "key", "hello"},
		{"invalid json", map[string]json.RawMessage{"key": json.RawMessage(`not-json`)}, "key", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extraString(tt.extra, tt.key)
			if got != tt.want {
				t.Errorf("extraString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJsonQuote(t *testing.T) {
	got := jsonQuote("hello world")
	if string(got) != `"hello world"` {
		t.Errorf("jsonQuote = %s, want %q", got, `"hello world"`)
	}
}
