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
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
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
	cfg := config.DefaultConfig()
	cfg.Mode = "auto"
	l := New(projectDir, "noodle", cfg, Dependencies{
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

func TestAutoMergeWithLocalChanges(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// fakeWorktree defaults to HasUnmergedCommits=true, triggering merge
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

func TestSupervisedMergeWithLocalChangesParksPendingReview(t *testing.T) {
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
		t.Fatalf("worktree merges = %d, want 0 (supervised mode requires review)", len(wt.merged))
	}
	pending, ok := l.cooks.pendingReview["42"]
	if !ok {
		t.Fatal("expected order 42 in pending review")
	}
	if pending.reason != "supervised mode requires merge approval" {
		t.Fatalf("pending review reason = %q, want %q", pending.reason, "supervised mode requires merge approval")
	}

	updatedOrders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updatedOrders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(updatedOrders.Orders))
	}
	if updatedOrders.Orders[0].Stages[0].Status != StageStatusActive {
		t.Fatalf("stage status = %q, want %q", updatedOrders.Orders[0].Stages[0].Status, StageStatusActive)
	}
}

func TestAutoAdvanceWithoutChanges(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	// In auto mode, stages without local or remote changes advance without merge.
	orders := OrdersFile{Orders: []Order{testOrder("42", "review", "review", "claude", "claude-opus-4-6")}}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Mode = "auto"

	rt := newMockRuntime()
	wt := &fakeWorktree{
		hasUnmergedCommits: map[string]bool{
			"42-0-review": false,
		},
	}
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
		t.Fatalf("worktree merges = %d, want 0 (no changes)", len(wt.merged))
	}
	// In auto mode, stages without changes advance without parking.
	if len(l.cooks.pendingReview) != 0 {
		t.Fatalf("pendingReview = %d, want 0 (auto mode advances stages without changes)", len(l.cooks.pendingReview))
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

func TestAutoMergeWithRemoteSyncResult(t *testing.T) {
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

	rt := newMockRuntime()
	wt := &fakeWorktree{
		hasUnmergedCommits: map[string]bool{
			"42-0-execute": false,
		},
	}
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
	if len(wt.remoteMerged) != 1 || wt.remoteMerged[0] != "noodle/session-a" {
		t.Fatalf("remote merges = %#v, want [noodle/session-a]", wt.remoteMerged)
	}
	if len(wt.merged) != 0 {
		t.Fatalf("worktree merges = %d, want 0 (remote merge path)", len(wt.merged))
	}
}

func TestStageCompletedPersistsCanonicalMergeRecovery(t *testing.T) {
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
	l.canonical = state.State{
		Orders:         map[string]state.OrderNode{},
		PendingReviews: map[string]state.PendingReviewNode{},
		Mode:           state.RunModeAuto,
		SchemaVersion:  statever.Current,
	}
	l.canonicalLoaded = true
	l.syncCanonicalOrderFromLegacy(orders.Orders[0])

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-1", stageIndex: 0},
		worktreeName: "order-1-0-execute",
		session:      &adoptedSession{id: "sess-1", status: "completed"},
	}

	if err := l.emitEventChecked(ingest.EventStageCompleted, l.mergeLifecyclePayload(cook, true)); err != nil {
		t.Fatalf("emit stage_completed: %v", err)
	}

	stage := l.canonical.Orders["order-1"].Stages[0]
	if stage.Status != state.StageMerging {
		t.Fatalf("status = %q, want %q", stage.Status, state.StageMerging)
	}
	if stage.Merge == nil {
		t.Fatal("expected canonical merge recovery")
	}
	if stage.Merge.WorktreeName != "order-1-0-execute" {
		t.Fatalf("merge worktree = %q, want %q", stage.Merge.WorktreeName, "order-1-0-execute")
	}
	if stage.Merge.Mode != "local" {
		t.Fatalf("merge mode = %q, want %q", stage.Merge.Mode, "local")
	}
	snapshot, err := reducer.ReadSnapshot(l.canonicalSnapshotPath())
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	snapStage := snapshot.State.Orders["order-1"].Stages[0]
	if snapStage.Merge == nil || snapStage.Merge.WorktreeName != "order-1-0-execute" {
		t.Fatalf("snapshot merge recovery = %#v", snapStage.Merge)
	}
}

func TestStageCompletedPersistsRemoteMergeRecovery(t *testing.T) {
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
	l.canonical = state.State{
		Orders:         map[string]state.OrderNode{},
		PendingReviews: map[string]state.PendingReviewNode{},
		Mode:           state.RunModeAuto,
		SchemaVersion:  statever.Current,
	}
	l.canonicalLoaded = true
	l.syncCanonicalOrderFromLegacy(orders.Orders[0])

	cook := &cookHandle{
		cookIdentity: cookIdentity{orderID: "order-r", stageIndex: 0},
		worktreeName: "order-r-0-execute",
		session:      &adoptedSession{id: "sess-r", status: "completed"},
	}
	sessionPath := filepath.Join(runtimeDir, "sessions", "sess-r")
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sessionPath, "spawn.json"),
		[]byte(`{"sync":{"type":"branch","branch":"noodle/remote-branch"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	if err := l.emitEventChecked(ingest.EventStageCompleted, l.mergeLifecyclePayload(cook, true)); err != nil {
		t.Fatalf("emit stage_completed: %v", err)
	}

	stage := l.canonical.Orders["order-r"].Stages[0]
	if stage.Merge == nil {
		t.Fatal("expected canonical merge recovery")
	}
	if stage.Merge.Mode != "remote" {
		t.Fatalf("merge mode = %q, want %q", stage.Merge.Mode, "remote")
	}
	if stage.Merge.Branch != "noodle/remote-branch" {
		t.Fatalf("merge branch = %q, want %q", stage.Merge.Branch, "noodle/remote-branch")
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
	l.canonical = state.State{
		Orders: map[string]state.OrderNode{
			"stuck-1": {
				OrderID:  "stuck-1",
				Status:   state.OrderActive,
				Stages:   []state.StageNode{{StageIndex: 0, Status: state.StageMerging, Skill: "execute", Runtime: "process"}},
				Metadata: map[string]string{},
			},
		},
		PendingReviews: map[string]state.PendingReviewNode{},
		Mode:           state.RunModeAuto,
		SchemaVersion:  statever.Current,
	}
	l.canonicalLoaded = true

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
	orders := OrdersFile{Orders: []Order{testOrder("adopted-1", "execute", "execute", "claude", "claude-opus-4-6")}}
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
	l.canonical = state.State{
		Orders: map[string]state.OrderNode{
			"adopted-1": {
				OrderID: "adopted-1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StageMerging,
					Skill:      "execute",
					Runtime:    "process",
					Merge: &state.MergeRecoveryNode{
						WorktreeName: "adopted-1-0-execute",
						Mode:         "local",
					},
				}},
			},
		},
		PendingReviews: map[string]state.PendingReviewNode{},
		Mode:           state.RunModeAuto,
		SchemaVersion:  statever.Current,
	}
	l.canonicalLoaded = true

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
	if stage := l.canonical.Orders["adopted-1"].Stages[0]; stage.Status != state.StageRunning || stage.Merge != nil {
		t.Fatalf("canonical stage after reconcile = %#v", stage)
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
		{"missing key", map[string]json.RawMessage{"other": json.RawMessage(`"val"`)}, "key", ""},
		{"present", map[string]json.RawMessage{"key": json.RawMessage(`"hello"`)}, "key", "hello"},
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
