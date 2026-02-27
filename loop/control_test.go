package loop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
)

func newControlTestLoop(t *testing.T, wt *fakeWorktree, sp *fakeDispatcher) *Loop {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: sp,
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID:      "42",
		stageIndex:   0,
		stage:        Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6"},
		worktreeName: "42-0-execute",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42-0-execute"),
		sessionID:    "sess-42",
	}
	if err := l.writePendingReview(); err != nil {
		t.Fatalf("write pending review: %v", err)
	}
	return l
}

func TestControlMergeKeepsPendingOnMergeFailure(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{mergeErr: errors.New("merge failed")}, &fakeDispatcher{})

	err := l.controlMerge("42")
	if err == nil {
		t.Fatal("expected merge error")
	}
	if _, ok := l.pendingReview["42"]; !ok {
		t.Fatal("pending review item should remain when merge fails")
	}
}

func TestControlMergeRemovesPendingAfterSuccess(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after successful merge")
	}
	items, err := ReadPendingReview(filepath.Join(l.projectDir, ".noodle"))
	if err != nil {
		t.Fatalf("read pending review: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("pending review file should be empty, got %d item(s)", len(items))
	}
}

func TestControlRejectRemovesPendingAfterSuccess(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	if err := l.controlReject("42"); err != nil {
		t.Fatalf("controlReject: %v", err)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after successful reject")
	}
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected rejected item to be tracked as failed")
	}
}

// controlRequestChanges now calls failStage (not dispatch). With no OnFailure stages,
// it should be terminal — the order is removed and markFailed is called.
func TestControlRequestChangesNoOnFailureTerminal(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	if err := l.controlRequestChanges("42", "Add missing tests"); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after request-changes")
	}
	// No OnFailure stages → terminal → markFailed.
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected order to be marked failed (no OnFailure stages)")
	}
	// Order should be removed from orders.json.
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	for _, o := range orders.Orders {
		if o.ID == "42" {
			t.Fatal("order 42 should be removed from orders.json")
		}
	}
}

// controlRequestChanges with OnFailure stages triggers failStage non-terminally.
func TestControlRequestChangesWithOnFailure(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages:    []Stage{{TaskKey: "execute", Status: StageStatusActive}},
		OnFailure: []Stage{{TaskKey: "debugging", Status: StageStatusPending}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID: "42", stageIndex: 0,
		stage:        Stage{TaskKey: "execute"},
		worktreeName: "42-0-execute",
	}

	if err := l.controlRequestChanges("42", "fix tests"); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review should be removed")
	}
	// OnFailure exists → not terminal → markFailed NOT called.
	if _, ok := l.failedTargets["42"]; ok {
		t.Fatal("order should not be in failedTargets (OnFailure exists)")
	}
	// Order should still be in orders.json with status "failing".
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	found := false
	for _, o := range orders.Orders {
		if o.ID == "42" {
			found = true
			if o.Status != OrderStatusFailing {
				t.Fatalf("order status = %q, want %q", o.Status, OrderStatusFailing)
			}
		}
	}
	if !found {
		t.Fatal("order 42 should still be in orders.json")
	}
}

func TestControlRequestChangesAllowsEmptyFeedback(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	if err := l.controlRequestChanges("42", "   "); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	// With empty feedback, order should still be terminally failed (no OnFailure).
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected order to be marked failed")
	}
}

func TestControlRequestChangesNotInPendingReview(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	err := l.controlRequestChanges("nonexistent", "feedback")
	if err == nil {
		t.Fatal("expected error for non-existent pending review item")
	}
	if !strings.Contains(err.Error(), "no pending review") {
		t.Fatalf("error = %q, want 'no pending review'", err.Error())
	}
}

func TestControlRejectKeepsPendingOnFailure(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	// Make the runtime dir unwritable so markFailed cannot write failed.json.
	if err := os.Chmod(l.runtimeDir, 0o444); err != nil {
		t.Fatalf("chmod runtime dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(l.runtimeDir, 0o755) })

	err := l.controlReject("42")
	if err == nil {
		t.Fatal("expected reject to fail when runtime dir is unwritable")
	}
	if _, ok := l.pendingReview["42"]; !ok {
		t.Fatal("pending review item should remain when reject fails")
	}
}

// --- New Phase 6 Tests ---

func TestControlEnqueueCreatesSingleStageOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	cmd := ControlCommand{
		Action:   "enqueue",
		OrderID:  "task-1",
		Prompt:   "Fix the login bug",
		TaskKey:  "execute",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Skill:    "execute",
	}
	if err := l.controlEnqueue(cmd); err != nil {
		t.Fatalf("controlEnqueue: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	o := orders.Orders[0]
	if o.ID != "task-1" {
		t.Fatalf("order ID = %q, want task-1", o.ID)
	}
	if o.Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", o.Status, OrderStatusActive)
	}
	if len(o.Stages) != 1 {
		t.Fatalf("stages count = %d, want 1", len(o.Stages))
	}
	s := o.Stages[0]
	if s.Status != StageStatusPending {
		t.Fatalf("stage status = %q, want %q", s.Status, StageStatusPending)
	}
	if s.Prompt != "Fix the login bug" {
		t.Fatalf("stage prompt = %q, want 'Fix the login bug'", s.Prompt)
	}
	if s.TaskKey != "execute" {
		t.Fatalf("stage task_key = %q, want execute", s.TaskKey)
	}
}

func TestControlEditItemModifiesStageFields(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Prompt: "old prompt", Provider: "claude", Model: "old-model", Skill: "old-skill", Status: StageStatusPending}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	cmd := ControlCommand{
		Action:   "edit-item",
		OrderID:  "42",
		Prompt:   "new prompt",
		TaskKey:  "quality",
		Provider: "codex",
		Model:    "gpt-5.3-codex",
		Skill:    "new-skill",
	}
	if err := l.controlEditItem(cmd); err != nil {
		t.Fatalf("controlEditItem: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	s := orders.Orders[0].Stages[0]
	if s.Prompt != "new prompt" {
		t.Fatalf("prompt = %q, want 'new prompt'", s.Prompt)
	}
	if s.TaskKey != "quality" {
		t.Fatalf("task_key = %q, want quality", s.TaskKey)
	}
	if s.Provider != "codex" {
		t.Fatalf("provider = %q, want codex", s.Provider)
	}
	if s.Model != "gpt-5.3-codex" {
		t.Fatalf("model = %q, want gpt-5.3-codex", s.Model)
	}
	if s.Skill != "new-skill" {
		t.Fatalf("skill = %q, want new-skill", s.Skill)
	}
}

func TestControlReorderChangesOrderPosition(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{
		{ID: "a", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
		{ID: "b", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
		{ID: "c", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
	}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Move "c" to position 0.
	cmd := ControlCommand{Action: "reorder", OrderID: "c", Value: "0"}
	if err := l.controlReorder(cmd); err != nil {
		t.Fatalf("controlReorder: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 3 {
		t.Fatalf("orders count = %d, want 3", len(orders.Orders))
	}
	ids := make([]string, len(orders.Orders))
	for i, o := range orders.Orders {
		ids[i] = o.ID
	}
	expected := "c,a,b"
	got := strings.Join(ids, ",")
	if got != expected {
		t.Fatalf("order after reorder = %q, want %q", got, expected)
	}
}

func TestControlSkipCancelsRemainingStages(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusCompleted},
			{TaskKey: "quality", Status: StageStatusPending},
			{TaskKey: "reflect", Status: StageStatusPending},
		},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.controlSkip("42"); err != nil {
		t.Fatalf("controlSkip: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	// Order should be removed after cancellation.
	for _, o := range orders.Orders {
		if o.ID == "42" {
			t.Fatal("order 42 should be removed from orders after skip")
		}
	}
}

func TestControlRejectSkipsOnFailure(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages:    []Stage{{TaskKey: "execute", Status: StageStatusActive}},
		OnFailure: []Stage{{TaskKey: "debugging", Status: StageStatusPending}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID: "42", stageIndex: 0,
		stage:        Stage{TaskKey: "execute"},
		worktreeName: "42-0-execute",
	}

	if err := l.controlReject("42"); err != nil {
		t.Fatalf("controlReject: %v", err)
	}
	// User rejection is terminal — skips OnFailure. Order removed, failure marked.
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected order to be in failed targets after reject")
	}
	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	for _, o := range orders.Orders {
		if o.ID == "42" {
			t.Fatal("order 42 should be removed (reject skips OnFailure)")
		}
	}
}

func TestControlRequeueResetsFailedOrderStages(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusFailing,
		Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusFailed},
			{TaskKey: "quality", Status: StageStatusCancelled},
		},
		OnFailure: []Stage{
			{TaskKey: "debugging", Status: StageStatusFailed},
		},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.failedTargets = map[string]string{"42": "test failure"}
	if err := l.writeFailedTargets(); err != nil {
		t.Fatalf("write failed targets: %v", err)
	}

	if err := l.controlRequeue("42"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}

	// Failed target should be removed.
	if _, ok := l.failedTargets["42"]; ok {
		t.Fatal("order 42 should not be in failed targets after requeue")
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	o := orders.Orders[0]
	if o.Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", o.Status, OrderStatusActive)
	}
	// All main stages should be reset to pending.
	for i, s := range o.Stages {
		if s.Status != StageStatusPending {
			t.Fatalf("Stages[%d].Status = %q, want pending", i, s.Status)
		}
	}
	// All OnFailure stages should also be reset to pending.
	for i, s := range o.OnFailure {
		if s.Status != StageStatusPending {
			t.Fatalf("OnFailure[%d].Status = %q, want pending", i, s.Status)
		}
	}
}

func TestControlRequeueFailingStatusResetsToActive(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusFailing,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusFailed}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.failedTargets = map[string]string{"42": "failed"}
	if err := l.writeFailedTargets(); err != nil {
		t.Fatalf("write failed targets: %v", err)
	}

	if err := l.controlRequeue("42"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}

	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if orders.Orders[0].Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", orders.Orders[0].Status, OrderStatusActive)
	}
}

func TestControlRequeueOrderNotInOrdersFile(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Empty orders — order "42" no longer exists.
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.failedTargets = map[string]string{"42": "failed"}
	if err := l.writeFailedTargets(); err != nil {
		t.Fatalf("write failed targets: %v", err)
	}

	// Should not error — just remove the failure marker.
	if err := l.controlRequeue("42"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}
	if _, ok := l.failedTargets["42"]; ok {
		t.Fatal("failure marker should be removed even if order is gone")
	}
}

func TestControlMergeChecksQualityVerdictReject(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusActive}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	// Write quality verdict that rejects.
	qualityDir := filepath.Join(runtimeDir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("mkdir quality: %v", err)
	}
	verdict := QualityVerdict{Accept: false, Feedback: "code coverage too low"}
	data, _ := json.Marshal(verdict)
	if err := os.WriteFile(filepath.Join(qualityDir, "sess-42.json"), data, 0o644); err != nil {
		t.Fatalf("write verdict: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID: "42", stageIndex: 0,
		stage:        Stage{TaskKey: "execute"},
		worktreeName: "42-0-execute",
		sessionID:    "sess-42",
	}

	// controlMerge should call failStage instead of merging.
	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	// No OnFailure → terminal → markFailed.
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected order to be marked failed (quality rejected)")
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review should be removed after quality rejection")
	}
}

func TestControlMergeNoVerdictProceeds(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, &fakeDispatcher{})

	// No quality verdict file at all — merge should proceed normally.
	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review should be removed after successful merge")
	}
	if _, ok := l.failedTargets["42"]; ok {
		t.Fatal("order should not be failed when no verdict file exists")
	}
}

func TestControlMergeFinalStageActiveOrderFiresDone(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Single-stage active order — final stage merge should fire adapter "done".
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusActive}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	ar := &fakeAdapterRunner{}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    ar,
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID: "42", stageIndex: 0,
		stage:        Stage{TaskKey: "execute"},
		worktreeName: "42-0-execute",
		sessionID:    "sess-42",
	}

	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	// Order was the final stage of an active order → advanceAndPersist fires adapter "done".
	if len(ar.doneCalls) == 0 {
		t.Fatal("expected adapter 'done' to be called for final stage of active order")
	}
	if ar.doneCalls[0] != "42" {
		t.Fatalf("done call arg = %q, want 42", ar.doneCalls[0])
	}
}

func TestControlMergeFinalOnFailureStageCallsMarkFailed(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Order in "failing" status with a single OnFailure stage.
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusFailing,
		Stages:    []Stage{{TaskKey: "execute", Status: StageStatusFailed}},
		OnFailure: []Stage{{TaskKey: "debugging", Status: StageStatusActive}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID: "42", stageIndex: 0,
		stage:        Stage{TaskKey: "debugging"},
		worktreeName: "42-0-debugging",
		sessionID:    "sess-42",
	}

	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	// Final OnFailure stage of failing order → markFailed (not adapter "done").
	if _, ok := l.failedTargets["42"]; !ok {
		t.Fatal("expected order to be marked failed after final OnFailure stage")
	}
}

func TestControlOrderIDFieldDecodedCorrectly(t *testing.T) {
	// Verify JSON decoding of the renamed field.
	raw := `{"id":"cmd-1","action":"merge","order_id":"42"}`
	var cmd ControlCommand
	if err := json.Unmarshal([]byte(raw), &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.OrderID != "42" {
		t.Fatalf("order_id = %q, want 42", cmd.OrderID)
	}
}

// --- Monotonic sequence tests ---

func TestCommandSequenceAssignment(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Write two control commands.
	controlPath := filepath.Join(runtimeDir, "control.ndjson")
	cmd1, _ := json.Marshal(ControlCommand{ID: "c1", Action: "pause"})
	cmd2, _ := json.Marshal(ControlCommand{ID: "c2", Action: "resume"})
	if err := os.WriteFile(controlPath, append(append(cmd1, '\n'), append(cmd2, '\n')...), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}

	if err := l.processControlCommands(); err != nil {
		t.Fatalf("processControlCommands: %v", err)
	}

	// Both commands should be processed; lastAppliedSeq should be 2.
	if l.lastAppliedSeq != 2 {
		t.Fatalf("lastAppliedSeq = %d, want 2", l.lastAppliedSeq)
	}
	if l.cmdSeqCounter != 2 {
		t.Fatalf("cmdSeqCounter = %d, want 2", l.cmdSeqCounter)
	}
	if _, ok := l.processedIDs["c1"]; !ok {
		t.Fatal("c1 should be in processedIDs")
	}
	if _, ok := l.processedIDs["c2"]; !ok {
		t.Fatal("c2 should be in processedIDs")
	}
}

func TestCommandSequencePersistenceRoundTrip(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Process a command so lastAppliedSeq is nonzero.
	controlPath := filepath.Join(runtimeDir, "control.ndjson")
	cmd, _ := json.Marshal(ControlCommand{ID: "c1", Action: "pause"})
	if err := os.WriteFile(controlPath, append(cmd, '\n'), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("processControlCommands: %v", err)
	}
	if l.lastAppliedSeq != 1 {
		t.Fatalf("lastAppliedSeq = %d, want 1", l.lastAppliedSeq)
	}

	// Persist via writeLastAppliedSeq.
	if err := l.writeLastAppliedSeq(); err != nil {
		t.Fatalf("writeLastAppliedSeq: %v", err)
	}

	// Create a new loop and hydrate — should recover the sequence.
	l2 := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	if err := l2.hydrateProcessedCommands(); err != nil {
		t.Fatalf("hydrateProcessedCommands: %v", err)
	}
	if l2.lastAppliedSeq != 1 {
		t.Fatalf("restored lastAppliedSeq = %d, want 1", l2.lastAppliedSeq)
	}
	if l2.cmdSeqCounter != 1 {
		t.Fatalf("restored cmdSeqCounter = %d, want 1", l2.cmdSeqCounter)
	}
}

func TestCommandSequenceSkipsReplayedCommands(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Simulate a restored sequence — commands with seq <= 5 were already applied.
	l.lastAppliedSeq = 5
	l.cmdSeqCounter = 5

	// Write a command that would get seq=6 (should be applied).
	controlPath := filepath.Join(runtimeDir, "control.ndjson")
	cmd, _ := json.Marshal(ControlCommand{ID: "new-cmd", Action: "pause"})
	if err := os.WriteFile(controlPath, append(cmd, '\n'), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("processControlCommands: %v", err)
	}

	// Should be applied, moving lastAppliedSeq to 6.
	if l.lastAppliedSeq != 6 {
		t.Fatalf("lastAppliedSeq = %d, want 6", l.lastAppliedSeq)
	}
	if l.state != StatePaused {
		t.Fatalf("state = %q, want paused", l.state)
	}
}

func TestCommandSequenceIdempotentReplay(t *testing.T) {
	// Simulate crash window: orders.json updated but last-applied-seq not yet written.
	// On restart, the same commands may be re-ingested. The ID-based dedup catches them
	// even if the sequence counter starts from a stale value.
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// First pass: process an enqueue command.
	controlPath := filepath.Join(runtimeDir, "control.ndjson")
	cmd, _ := json.Marshal(ControlCommand{ID: "enq-1", Action: "enqueue", OrderID: "order-1", Prompt: "test"})
	if err := os.WriteFile(controlPath, append(cmd, '\n'), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("first processControlCommands: %v", err)
	}
	orders, _ := l.currentOrders()
	if len(orders.Orders) != 1 {
		t.Fatalf("orders after first pass = %d, want 1", len(orders.Orders))
	}

	// Re-inject the same command (simulates incomplete truncate on crash).
	if err := os.WriteFile(controlPath, append(cmd, '\n'), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("second processControlCommands: %v", err)
	}
	orders, _ = l.currentOrders()
	if len(orders.Orders) != 1 {
		t.Fatalf("orders after replay = %d, want 1 (idempotent)", len(orders.Orders))
	}
}

func TestFlushStatePersistsLastAppliedSeq(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Process a command.
	controlPath := filepath.Join(runtimeDir, "control.ndjson")
	cmd, _ := json.Marshal(ControlCommand{ID: "c1", Action: "resume"})
	if err := os.WriteFile(controlPath, append(cmd, '\n'), 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.processControlCommands(); err != nil {
		t.Fatalf("processControlCommands: %v", err)
	}

	// Flush state.
	if err := l.flushState(); err != nil {
		t.Fatalf("flushState: %v", err)
	}

	// Verify the file was written.
	data, err := os.ReadFile(filepath.Join(runtimeDir, "last-applied-seq"))
	if err != nil {
		t.Fatalf("read last-applied-seq: %v", err)
	}
	if strings.TrimSpace(string(data)) != "1" {
		t.Fatalf("last-applied-seq file = %q, want '1'", strings.TrimSpace(string(data)))
	}
}

func TestFailedTargetStickiness(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Create an order with the same ID as a failed target.
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
	}}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.failedTargets = map[string]string{"42": "previous failure"}

	// Dispatching should skip this order due to failed target stickiness.
	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	failedSet := make(map[string]struct{})
	for id := range l.failedTargets {
		failedSet[id] = struct{}{}
	}
	candidates := dispatchableStages(orders, nil, failedSet, nil, nil)
	if len(candidates) != 0 {
		t.Fatal("failed target should block dispatch of order with same ID")
	}

	// After requeue, the order should be dispatchable.
	delete(l.failedTargets, "42")
	candidates = dispatchableStages(orders, nil, nil, nil, nil)
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1 after requeue", len(candidates))
	}
}
