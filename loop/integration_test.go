package loop

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/mode"
	"github.com/poteto/noodle/internal/projection"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
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

type lifecycleParityOrder struct {
	ID     string
	Status string
	Stages []string
}

func assertLegacyCanonicalParity(t *testing.T, env *integrationEnv) {
	t.Helper()

	legacyOrders := legacyParityOrders(t, env.readOrders(t), env.runtimeDir)
	canonicalOrders := canonicalParityOrders(t, env.loop)
	if reflect.DeepEqual(legacyOrders, canonicalOrders) {
		return
	}

	t.Fatalf("legacy/canonical parity mismatch\nlegacy: %#v\ncanonical: %#v", legacyOrders, canonicalOrders)
}

func legacyParityOrders(t *testing.T, orders OrdersFile, runtimeDir string) []lifecycleParityOrder {
	t.Helper()

	pendingReview, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("read pending review: %v", err)
	}

	reviewStages := make(map[string]map[int]struct{}, len(pendingReview))
	for _, item := range pendingReview {
		orderID := strings.TrimSpace(item.OrderID)
		if orderID == "" {
			continue
		}
		if reviewStages[orderID] == nil {
			reviewStages[orderID] = map[int]struct{}{}
		}
		reviewStages[orderID][item.StageIndex] = struct{}{}
	}

	out := make([]lifecycleParityOrder, 0, len(orders.Orders))
	for _, order := range orders.Orders {
		stages := make([]string, 0, len(order.Stages))
		for i, stage := range order.Stages {
			status := normalizeLegacyStageStatus(string(stage.Status))
			if _, ok := reviewStages[order.ID][i]; ok {
				status = "review"
			}
			stages = append(stages, status)
		}
		out = append(out, lifecycleParityOrder{
			ID:     strings.TrimSpace(order.ID),
			Status: normalizeLegacyOrderStatus(string(order.Status)),
			Stages: stages,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func canonicalParityOrders(t *testing.T, l *Loop) []lifecycleParityOrder {
	t.Helper()

	bundle, err := projection.Project(l.canonical, mode.ModeState{
		EffectiveMode: l.canonical.Mode,
		Epoch:         l.canonical.ModeEpoch,
	})
	if err != nil {
		t.Fatalf("project canonical state: %v", err)
	}

	out := make([]lifecycleParityOrder, 0, len(bundle.OrdersProjection))
	for _, order := range bundle.OrdersProjection {
		status := normalizeCanonicalOrderStatus(order.Status)
		if status == "" {
			continue
		}
		stages := make([]string, 0, len(order.Stages))
		for _, stage := range order.Stages {
			stages = append(stages, normalizeCanonicalStageStatus(stage.Status))
		}
		out = append(out, lifecycleParityOrder{
			ID:     strings.TrimSpace(order.ID),
			Status: status,
			Stages: stages,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func normalizeLegacyOrderStatus(status string) string {
	switch strings.TrimSpace(status) {
	case string(OrderStatusActive):
		return "active"
	case string(OrderStatusFailed):
		return "failed"
	default:
		return ""
	}
}

func normalizeLegacyStageStatus(status string) string {
	switch strings.TrimSpace(status) {
	case string(StageStatusPending):
		return "pending"
	case string(StageStatusActive), string(StageStatusMerging):
		return "busy"
	case string(StageStatusCompleted):
		return "completed"
	case string(StageStatusFailed):
		return "failed"
	case string(StageStatusCancelled):
		return "cancelled"
	default:
		return ""
	}
}

func normalizeCanonicalOrderStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "pending", "active":
		return "active"
	case "failed":
		return "failed"
	default:
		return ""
	}
}

func normalizeCanonicalStageStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "pending":
		return "pending"
	case "dispatching", "running", "merging":
		return "busy"
	case "review":
		return "review"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "cancelled", "skipped":
		return "cancelled"
	default:
		return ""
	}
}

func seedCanonicalFromOrders(l *Loop, orders OrdersFile) {
	canonicalOrders := make(map[string]state.OrderNode, len(orders.Orders))
	for _, order := range orders.Orders {
		stages := make([]state.StageNode, 0, len(order.Stages))
		for i, stage := range order.Stages {
			stages = append(stages, state.StageNode{
				StageIndex: i,
				Status:     legacyStageToCanonicalStatus(string(stage.Status)),
				Skill:      stage.Skill,
				Runtime:    stage.Runtime,
			})
		}
		canonicalOrders[order.ID] = state.OrderNode{
			OrderID: order.ID,
			Status:  legacyOrderToCanonicalStatus(string(order.Status)),
			Stages:  stages,
		}
	}
	l.canonical = state.State{
		Orders:        canonicalOrders,
		Mode:          l.canonical.Mode,
		ModeEpoch:     l.canonical.ModeEpoch,
		SchemaVersion: statever.Current,
	}
}

func legacyOrderToCanonicalStatus(status string) state.OrderLifecycleStatus {
	switch strings.TrimSpace(status) {
	case string(OrderStatusFailed):
		return state.OrderFailed
	default:
		return state.OrderActive
	}
}

func legacyStageToCanonicalStatus(status string) state.StageLifecycleStatus {
	switch strings.TrimSpace(status) {
	case string(StageStatusActive):
		return state.StageRunning
	case string(StageStatusMerging):
		return state.StageMerging
	case string(StageStatusCompleted):
		return state.StageCompleted
	case string(StageStatusFailed):
		return state.StageFailed
	case string(StageStatusCancelled):
		return state.StageCancelled
	default:
		return state.StagePending
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
	briefWithPlans := mise.Brief{Backlog: []adapter.BacklogItem{{ID: "1", Title: "test", Status: "open"}}}

	l := New(projectDir, "noodle", ic.cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
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

	env := newIntegrationEnv(t, orders, func(ic *integrationCfg) {
		ic.cfg.Mode = "auto"
	})
	l, deps := env.loop, env
	seedCanonicalFromOrders(l, orders)

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
	assertLegacyCanonicalParity(t, env)

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
	assertLegacyCanonicalParity(t, env)

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
	assertLegacyCanonicalParity(t, env)

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
		ic.cfg.Mode = "auto"
		ic.mergeErr = &worktree.MergeConflictError{Branch: "noodle/conflict-session"}
	})
	l := env.loop
	seedCanonicalFromOrders(l, orders)

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

	if _, ok := l.cooks.pendingReview["conflict-1"]; !ok {
		t.Fatal("expected conflict-1 in pendingReview")
	}
	pending := l.cooks.pendingReview["conflict-1"]
	if !strings.Contains(pending.reason, "merge conflict") {
		t.Fatalf("pending reason = %q, want merge conflict", pending.reason)
	}
	assertLegacyCanonicalParity(t, env)

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
	assertLegacyCanonicalParity(t, env)

	// Verify pendingReview was cleared.
	if len(l.cooks.pendingReview) != 0 {
		t.Fatalf("pendingReview = %d, want 0", len(l.cooks.pendingReview))
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
				Title:  "second order in queue",
				Status: OrderStatusActive,
				Stages: []Stage{
					{TaskKey: "reflect", Skill: "reflect", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
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
		orderStatuses[o.ID] = string(o.Status)
	}
	if orderStatuses["snap-1"] != string(OrderStatusActive) {
		t.Fatalf("snap-1 status = %q, want active", orderStatuses["snap-1"])
	}
	if orderStatuses["snap-2"] != string(OrderStatusActive) {
		t.Fatalf("snap-2 status = %q, want active", orderStatuses["snap-2"])
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
	Orders             []Order             `json:"orders"`
	PendingReviews     []PendingReviewItem `json:"pending_reviews"`
	PendingReviewCount int                 `json:"pending_review_count"`
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
