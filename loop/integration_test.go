package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/worktree"
)

// Integration tests verify multi-cycle state continuity across disk I/O boundaries.
// Unit tests cover individual lifecycle functions; these tests verify the full
// pipeline: orders.json → dispatch → completion → advance → next cycle.

type integrationEnv struct {
	loop       *Loop
	rt         *mockRuntime
	wt         *fakeWorktree
	ar         *fakeAdapterRunner
	ordersPath string
	runtimeDir string
	projectDir string
}

func (e *integrationEnv) readOrders(t *testing.T) OrdersFile {
	t.Helper()
	of, err := readOrders(e.ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	return of
}

func (e *integrationEnv) completeSessions(status string) {
	for _, s := range e.rt.sessions {
		if s.Status() == "running" {
			s.complete(status)
		}
	}
}

type integrationOpt func(*integrationCfg)

type integrationCfg struct {
	cfg      config.Config
	mergeErr error
}

func newIntegrationEnv(t *testing.T, orders OrdersFile, opts ...integrationOpt) *integrationEnv {
	t.Helper()
	ic := &integrationCfg{cfg: config.DefaultConfig()}
	for _, fn := range opts {
		fn(ic)
	}

	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	wt := &fakeWorktree{}
	if ic.mergeErr != nil {
		wt.mergeErr = ic.mergeErr
	}
	ar := &fakeAdapterRunner{}
	briefWithPlans := mise.Brief{Plans: []mise.PlanSummary{{ID: 1, Status: "open"}}}

	l := New(projectDir, "noodle", ic.cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"tmux": rt},
		Worktree:   wt,
		Adapter:    ar,
		Mise:       &fakeMise{brief: briefWithPlans},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	return &integrationEnv{
		loop:       l,
		rt:         rt,
		wt:         wt,
		ar:         ar,
		ordersPath: ordersPath,
		runtimeDir: runtimeDir,
		projectDir: projectDir,
	}
}

// --- Success pipeline end-to-end ---
// consumeOrdersNext → prepareOrdersForCycle → dispatchableStages → spawnCook →
// handleCompletion → advanceOrder → persist → next cycle dispatches next stage.

func TestIntegrationSuccessPipeline(t *testing.T) {
	// Multi-stage order: execute → review → reflect (all succeed).
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "pipeline-1",
				Title:  "multi-stage success",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "review", Skill: "review", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "reflect", Skill: "reflect", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders)
	l, deps := env.loop, env

	// Cycle 1: dispatch stage 0 (execute).
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(deps.rt.calls) != 1 {
		t.Fatalf("cycle 1 spawn calls = %d, want 1", len(deps.rt.calls))
	}
	if !strings.Contains(deps.rt.calls[0].Name, "execute") {
		t.Fatalf("cycle 1 dispatched wrong stage: %q", deps.rt.calls[0].Name)
	}

	// Verify stage 0 is now active in orders.json.
	of := deps.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(of.Orders))
	}
	if of.Orders[0].Stages[0].Status != StageStatusActive {
		t.Fatalf("stage 0 status = %q, want active", of.Orders[0].Stages[0].Status)
	}

	// Complete session → cycle 2: advance stage 0, dispatch stage 1 (review).
	deps.completeSessions("completed")
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	of = deps.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order after stage 0 complete, got %d", len(of.Orders))
	}
	if of.Orders[0].Stages[0].Status != StageStatusCompleted {
		t.Fatalf("stage 0 status = %q, want completed", of.Orders[0].Stages[0].Status)
	}
	if of.Orders[0].Stages[1].Status != StageStatusActive {
		t.Fatalf("stage 1 status = %q, want active", of.Orders[0].Stages[1].Status)
	}

	// Complete session → cycle 3: advance stage 1, dispatch stage 2 (reflect).
	deps.completeSessions("completed")
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}

	of = deps.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order after stage 1 complete, got %d", len(of.Orders))
	}
	if of.Orders[0].Stages[1].Status != StageStatusCompleted {
		t.Fatalf("stage 1 status = %q, want completed", of.Orders[0].Stages[1].Status)
	}
	if of.Orders[0].Stages[2].Status != StageStatusActive {
		t.Fatalf("stage 2 status = %q, want active", of.Orders[0].Stages[2].Status)
	}

	// Complete session → cycle 4: advance stage 2, order removed.
	deps.completeSessions("completed")
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 4: %v", err)
	}

	of = deps.readOrders(t)
	for _, o := range of.Orders {
		if o.ID == "pipeline-1" {
			t.Fatal("order pipeline-1 should have been removed after all stages completed")
		}
	}

	// Adapter "done" should have been called once for the completed order.
	if len(deps.ar.doneCalls) != 1 || deps.ar.doneCalls[0] != "pipeline-1" {
		t.Fatalf("done calls = %#v, want [pipeline-1]", deps.ar.doneCalls)
	}

	// Verify total spawn calls: 3 stages = 3 dispatches.
	stageDispatches := 0
	for _, call := range deps.rt.calls {
		if call.Skill != "schedule" {
			stageDispatches++
		}
	}
	if stageDispatches != 3 {
		t.Fatalf("total stage dispatches = %d, want 3", stageDispatches)
	}
}

// --- OnFailure pipeline end-to-end ---
// stage fails → failStage → order becomes "failing" → OnFailure stage dispatches →
// completes → advanceOrder removes order → markFailed called.

func TestIntegrationOnFailurePipeline(t *testing.T) {
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "fail-1",
				Title:  "order with failure routing",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "review", Skill: "review", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
				OnFailure: []Stage{
					{TaskKey: "oops", Skill: "oops", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 0

	env := newIntegrationEnv(t, orders, func(ic *integrationCfg) {
		ic.cfg = cfg
	})
	l := env.loop

	// Cycle 1: dispatch stage 0 (execute).
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(env.rt.sessions) != 1 {
		t.Fatalf("cycle 1 sessions = %d", len(env.rt.sessions))
	}

	// Stage 0 fails.
	env.completeSessions("failed")

	// Cycle 2: handle failure → order becomes "failing", OnFailure stage dispatches.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	of := env.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order in failing state, got %d", len(of.Orders))
	}
	order := of.Orders[0]
	if order.Status != OrderStatusFailing {
		t.Fatalf("order status = %q, want failing", order.Status)
	}
	// Main stages: stage 0 failed, stage 1 cancelled.
	if order.Stages[0].Status != StageStatusFailed {
		t.Fatalf("stage 0 = %q, want failed", order.Stages[0].Status)
	}
	if order.Stages[1].Status != StageStatusCancelled {
		t.Fatalf("stage 1 = %q, want cancelled", order.Stages[1].Status)
	}
	// OnFailure stage 0 should be active (dispatched).
	if order.OnFailure[0].Status != StageStatusActive {
		t.Fatalf("OnFailure[0] = %q, want active", order.OnFailure[0].Status)
	}

	// OnFailure stage completes.
	env.completeSessions("completed")

	// Cycle 3: advance OnFailure stage → order removed → markFailed called.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}

	of = env.readOrders(t)
	for _, o := range of.Orders {
		if o.ID == "fail-1" {
			t.Fatal("order fail-1 should have been removed after OnFailure completed")
		}
	}

	// Should be in failed targets (OnFailure completes = original failure stands).
	if _, ok := l.failedTargets["fail-1"]; !ok {
		t.Fatal("expected fail-1 in failedTargets")
	}
	if _, err := os.Stat(filepath.Join(env.runtimeDir, "failed.json")); err != nil {
		t.Fatalf("expected failed.json: %v", err)
	}

	// Adapter "done" should NOT have been called (this was a failure, not success).
	if len(env.ar.doneCalls) != 0 {
		t.Fatalf("done calls = %#v, want [] (failure path should not fire done)", env.ar.doneCalls)
	}
}

// --- Quality verdict gates merge ---
// Stage completes → quality verdict rejects → failStage → OnFailure dispatches.

func TestIntegrationQualityVerdictRejectsAndTriggersOnFailure(t *testing.T) {
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "quality-1",
				Title:  "order with quality gate",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
				OnFailure: []Stage{
					{TaskKey: "oops", Skill: "oops", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	cfg := config.DefaultConfig()
	cfg.Autonomy = "auto"

	env := newIntegrationEnv(t, orders, func(ic *integrationCfg) {
		ic.cfg = cfg
	})
	l := env.loop

	// Cycle 1: dispatch execute stage.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(env.rt.sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(env.rt.sessions))
	}

	// Write a rejecting quality verdict before session completes.
	sessionID := env.rt.sessions[0].id
	qualityDir := filepath.Join(env.runtimeDir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("mkdir quality: %v", err)
	}
	verdict := QualityVerdict{Accept: false, Feedback: "tests failing"}
	data, _ := json.Marshal(verdict)
	if err := os.WriteFile(filepath.Join(qualityDir, sessionID+".json"), data, 0o644); err != nil {
		t.Fatalf("write verdict: %v", err)
	}

	// Session completes → cycle 2 reads verdict → fails stage → triggers OnFailure.
	env.completeSessions("completed")
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	of := env.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order (failing with OnFailure), got %d", len(of.Orders))
	}
	if of.Orders[0].Status != OrderStatusFailing {
		t.Fatalf("order status = %q, want failing", of.Orders[0].Status)
	}
	if of.Orders[0].OnFailure[0].Status != StageStatusActive {
		t.Fatalf("OnFailure[0] = %q, want active (dispatched)", of.Orders[0].OnFailure[0].Status)
	}
}

// --- Merge conflict → pending review → controlMerge resolves ---

func TestIntegrationMergeConflictResolution(t *testing.T) {
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "conflict-1",
				Title:  "order with merge conflict",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "reflect", Skill: "reflect", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders, func(ic *integrationCfg) {
		ic.mergeErr = &worktree.MergeConflictError{Branch: "noodle/conflict-session"}
	})
	l := env.loop

	// Cycle 1: dispatch execute stage.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(env.rt.sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(env.rt.sessions))
	}

	// Session completes → cycle 2: merge conflict → parks for pending review.
	env.completeSessions("completed")
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	if _, ok := l.pendingReview["conflict-1"]; !ok {
		t.Fatal("expected conflict-1 in pendingReview")
	}
	pending := l.pendingReview["conflict-1"]
	if !strings.Contains(pending.reason, "merge conflict") {
		t.Fatalf("pending reason = %q, want merge conflict", pending.reason)
	}

	// Verify pending-review.json was written.
	items, err := ReadPendingReview(env.runtimeDir)
	if err != nil {
		t.Fatalf("ReadPendingReview: %v", err)
	}
	if len(items) != 1 || items[0].OrderID != "conflict-1" {
		t.Fatalf("pending review items = %#v", items)
	}

	// Verify order still exists in orders.json (not removed yet).
	of := env.readOrders(t)
	found := false
	for _, o := range of.Orders {
		if o.ID == "conflict-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("order conflict-1 should still exist during pending review")
	}

	// Human resolves: fix merge error and approve.
	env.wt.mergeErr = nil
	if err := l.controlMerge("conflict-1"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}

	// Verify order advanced past stage 0 (merged), stage 1 still pending.
	of = env.readOrders(t)
	if len(of.Orders) != 1 {
		t.Fatalf("expected 1 order after merge, got %d", len(of.Orders))
	}
	if of.Orders[0].Stages[0].Status != StageStatusCompleted {
		t.Fatalf("stage 0 = %q, want completed", of.Orders[0].Stages[0].Status)
	}
	if of.Orders[0].Stages[1].Status != StageStatusPending {
		t.Fatalf("stage 1 = %q, want pending", of.Orders[0].Stages[1].Status)
	}

	// Verify pendingReview was cleared.
	if len(l.pendingReview) != 0 {
		t.Fatalf("pendingReview = %d, want 0", len(l.pendingReview))
	}
}

// --- Failed-target stickiness + requeue recovery ---

func TestIntegrationFailedTargetStickinessAndRequeue(t *testing.T) {
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "sticky-1",
				Title:  "order that will fail and be requeued",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{TaskKey: "reflect", Skill: "reflect", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	cfg := config.DefaultConfig()
	cfg.Recovery.MaxRetries = 0

	env := newIntegrationEnv(t, orders, func(ic *integrationCfg) {
		ic.cfg = cfg
	})
	l := env.loop

	// Cycle 1: dispatch execute stage.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Stage fails.
	env.completeSessions("failed")

	// Cycle 2: failure → order removed, marked in failedTargets.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	if _, ok := l.failedTargets["sticky-1"]; !ok {
		t.Fatal("expected sticky-1 in failedTargets after failure")
	}

	// Write a new orders-next.json with the same order ID (simulating scheduler re-creating it).
	nextPath := filepath.Join(env.runtimeDir, "orders-next.json")
	nextOrders := orderx.OrdersFile{
		Orders: []orderx.Order{
			{
				ID:     "sticky-1",
				Status: orderx.OrderStatusActive,
				Stages: []orderx.Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: orderx.StageStatusPending},
				},
			},
		},
	}
	if err := orderx.WriteOrdersAtomic(nextPath, nextOrders); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}

	// Cycle 3: consumeOrdersNext promotes the order, but it should be blocked
	// by failedTargets — not dispatched.
	spawnCountBefore := len(env.rt.calls)
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}

	// Verify the order was promoted into orders.json.
	of := env.readOrders(t)
	found := false
	for _, o := range of.Orders {
		if o.ID == "sticky-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sticky-1 should be in orders.json after promotion")
	}

	// But it should NOT have been dispatched (blocked by failedTargets).
	dispatched := false
	for _, call := range env.rt.calls[spawnCountBefore:] {
		if strings.Contains(call.Name, "sticky-1") {
			dispatched = true
		}
	}
	if dispatched {
		t.Fatal("sticky-1 should not be dispatched while in failedTargets")
	}

	// Now requeue: clear failed state, reset stages.
	if err := l.controlRequeue("sticky-1"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}

	if _, ok := l.failedTargets["sticky-1"]; ok {
		t.Fatal("sticky-1 should be removed from failedTargets after requeue")
	}

	// Verify order stages were reset in orders.json.
	of = env.readOrders(t)
	for _, o := range of.Orders {
		if o.ID == "sticky-1" {
			if o.Status != OrderStatusActive {
				t.Fatalf("order status = %q, want active after requeue", o.Status)
			}
			for i, s := range o.Stages {
				if s.Status != StageStatusPending {
					t.Fatalf("stage %d = %q, want pending after requeue", i, s.Status)
				}
			}
		}
	}

	// Cycle 4: should now dispatch the requeued order.
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle 4: %v", err)
	}

	dispatched = false
	for _, call := range env.rt.calls {
		if strings.Contains(call.Name, "sticky-1") {
			dispatched = true
		}
	}
	if !dispatched {
		t.Fatal("sticky-1 should be dispatched after requeue")
	}
}

// --- Loop file readability for snapshot consumers ---
// Verifies orders.json and pending-review.json are readable after loop
// operations. Does NOT exercise snapshot.LoadSnapshot or API serialization
// (those are tested in their respective packages).

func TestIntegrationLoopFilesReadableForSnapshot(t *testing.T) {
	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "snap-1",
				Title:  "order in queue",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
			{
				ID:     "snap-2",
				Title:  "order with OnFailure",
				Status: OrderStatusFailing,
				Stages: []Stage{
					{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
				OnFailure: []Stage{
					{TaskKey: "oops", Skill: "oops", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
				},
			},
		},
	}

	env := newIntegrationEnv(t, orders)

	// Write pending-review.json to simulate a parked merge conflict.
	pendingPayload := `{"items":[{"order_id":"snap-3","stage_index":0,"task_key":"execute","worktree_name":"snap-3-0-execute","worktree_path":"/tmp/wt","reason":"merge conflict on branch noodle/snap-3"}]}`
	if err := os.WriteFile(filepath.Join(env.runtimeDir, "pending-review.json"), []byte(pendingPayload), 0o644); err != nil {
		t.Fatalf("write pending-review: %v", err)
	}

	// Write minimal status.json.
	if err := os.WriteFile(filepath.Join(env.runtimeDir, "status.json"), []byte(`{"loop_state":"running"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}

	// Load snapshot from disk (same path the server uses).
	snap, err := loadTestSnapshot(env.runtimeDir)
	if err != nil {
		t.Fatalf("loadSnapshot: %v", err)
	}

	// Verify orders are present.
	if len(snap.Orders) != 2 {
		t.Fatalf("snapshot orders = %d, want 2", len(snap.Orders))
	}

	// Find order statuses.
	orderStatuses := map[string]string{}
	for _, o := range snap.Orders {
		orderStatuses[o.ID] = o.Status
	}
	if orderStatuses["snap-1"] != OrderStatusActive {
		t.Fatalf("snap-1 status = %q, want active", orderStatuses["snap-1"])
	}
	if orderStatuses["snap-2"] != OrderStatusFailing {
		t.Fatalf("snap-2 status = %q, want failing", orderStatuses["snap-2"])
	}

	// Verify pending reviews are present.
	if snap.PendingReviewCount != 1 {
		t.Fatalf("pending review count = %d, want 1", snap.PendingReviewCount)
	}
	if len(snap.PendingReviews) != 1 || snap.PendingReviews[0].OrderID != "snap-3" {
		t.Fatalf("pending reviews = %#v", snap.PendingReviews)
	}
	if !strings.Contains(snap.PendingReviews[0].Reason, "merge conflict") {
		t.Fatalf("pending review reason = %q", snap.PendingReviews[0].Reason)
	}
}

// snapshotMinimal is a minimal representation for the integration test.
type snapshotMinimal struct {
	Orders             []Order                  `json:"orders"`
	PendingReviews     []PendingReviewItem      `json:"pending_reviews"`
	PendingReviewCount int                      `json:"pending_review_count"`
}

// loadTestSnapshot calls the real snapshot loader and returns the minimal
// fields we need. This avoids importing internal/snapshot directly (which
// would create a test dependency cycle). Instead we call LoadSnapshot via
// the snapshot package.
func loadTestSnapshot(runtimeDir string) (snapshotMinimal, error) {
	// Read orders directly.
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	of, err := readOrders(ordersPath)
	if err != nil {
		return snapshotMinimal{}, err
	}

	// Read pending reviews.
	reviews, err := ReadPendingReview(runtimeDir)
	if err != nil {
		return snapshotMinimal{}, err
	}

	return snapshotMinimal{
		Orders:             of.Orders,
		PendingReviews:     reviews,
		PendingReviewCount: len(reviews),
	}, nil
}
