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
	"github.com/poteto/noodle/internal/failure"
	loopruntime "github.com/poteto/noodle/runtime"
)

func newControlTestLoop(t *testing.T, wt *fakeWorktree, rt *mockRuntime) *Loop {
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
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   wt,
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.cooks.pendingReview["42"] = &pendingReviewCook{
		cookIdentity: cookIdentity{
			orderID:    "42",
			stageIndex: 0,
			stage:      Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6"},
		},
		worktreeName: "42-0-execute",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42-0-execute"),
		sessionID:    "sess-42",
	}
	if err := l.writePendingReview(); err != nil {
		t.Fatalf("write pending review: %v", err)
	}
	return l
}

func TestControlMergeMarksOrderFailedOnMergeFailure(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{mergeErr: errors.New("merge failed")}, newMockRuntime())

	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	if _, ok := l.cooks.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed when merge fails")
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	if orders.Orders[0].Status != OrderStatusFailed {
		t.Fatalf("order status = %q, want %q", orders.Orders[0].Status, OrderStatusFailed)
	}
	if got := orders.Orders[0].Stages[0].Status; got != StageStatusFailed {
		t.Fatalf("stage status = %q, want %q", got, StageStatusFailed)
	}
}

func TestControlMergeRemovesPendingAfterSuccess(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	if err := l.controlMerge("42"); err != nil {
		t.Fatalf("controlMerge: %v", err)
	}
	if _, ok := l.cooks.pendingReview["42"]; ok {
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
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	if err := l.controlReject("42"); err != nil {
		t.Fatalf("controlReject: %v", err)
	}
	if _, ok := l.cooks.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after successful reject")
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	for _, o := range orders.Orders {
		if o.ID == "42" {
			t.Fatal("order 42 should be removed from orders.json after reject")
		}
	}

	envelope := requireLastLoopFailureEnvelope(t, l)
	if envelope.Class != CycleFailureClassOrderHard {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassOrderHard)
	}
	if envelope.OrderClass != OrderFailureClassOrderTerminal {
		t.Fatalf("order class = %q, want %q", envelope.OrderClass, OrderFailureClassOrderTerminal)
	}
	if envelope.AgentMistake == nil {
		t.Fatal("reject should classify cook agent mistake")
	}
	if envelope.AgentMistake.Owner != failure.FailureOwnerCookAgent {
		t.Fatalf("owner = %q, want %q", envelope.AgentMistake.Owner, failure.FailureOwnerCookAgent)
	}
	if envelope.AgentMistake.CookReason != CookMistakeReasonReviewRejected {
		t.Fatalf("cook reason = %q, want %q", envelope.AgentMistake.CookReason, CookMistakeReasonReviewRejected)
	}
	if envelope.AgentMistake.Scope != failure.FailureScopeOrder {
		t.Fatalf("scope = %q, want %q", envelope.AgentMistake.Scope, failure.FailureScopeOrder)
	}
	if envelope.AgentMistake.OrderID != "42" {
		t.Fatalf("agent mistake order_id = %q, want 42", envelope.AgentMistake.OrderID)
	}
	if envelope.AgentMistake.StageIndex == nil || *envelope.AgentMistake.StageIndex != 0 {
		t.Fatalf("agent mistake stage_index = %v, want 0", envelope.AgentMistake.StageIndex)
	}

	events := readNDJSON(t, filepath.Join(l.runtimeDir, "loop-events.ndjson"))
	stageFailed := findEvents(events, LoopEventStageFailed)
	if len(stageFailed) == 0 {
		t.Fatal("expected stage.failed event after reject")
	}
	var stagePayload StageFailedPayload
	if err := json.Unmarshal(stageFailed[len(stageFailed)-1].Payload, &stagePayload); err != nil {
		t.Fatalf("parse stage.failed payload: %v", err)
	}
	if stagePayload.AgentMistake == nil || stagePayload.AgentMistake.Owner != failure.FailureOwnerCookAgent {
		t.Fatal("stage.failed payload missing cook ownership classification")
	}
	if stagePayload.Failure == nil {
		t.Fatal("stage.failed payload missing failure classification")
	}
	if stagePayload.Failure.Class != failure.FailureClassAgentMistake {
		t.Fatalf("failure class = %q, want %q", stagePayload.Failure.Class, failure.FailureClassAgentMistake)
	}
}

// controlRequestChanges should terminally fail the order and keep it for explicit requeue.
func TestControlRequestChangesMarksOrderFailed(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	if err := l.controlRequestChanges("42", "Add missing tests"); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	if _, ok := l.cooks.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after request-changes")
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	for _, o := range orders.Orders {
		if o.ID == "42" {
			if o.Status != OrderStatusFailed {
				t.Fatalf("order status = %q, want %q", o.Status, OrderStatusFailed)
			}
			if got := o.Stages[0].Status; got != StageStatusFailed {
				t.Fatalf("stage status = %q, want %q", got, StageStatusFailed)
			}
		}
	}

	envelope := requireLastLoopFailureEnvelope(t, l)
	if envelope.Class != CycleFailureClassOrderHard {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassOrderHard)
	}
	if envelope.OrderClass != OrderFailureClassStageTerminal {
		t.Fatalf("order class = %q, want %q", envelope.OrderClass, OrderFailureClassStageTerminal)
	}
	if envelope.AgentMistake == nil {
		t.Fatal("request-changes should classify cook agent mistake")
	}
	if envelope.AgentMistake.Owner != failure.FailureOwnerCookAgent {
		t.Fatalf("owner = %q, want %q", envelope.AgentMistake.Owner, failure.FailureOwnerCookAgent)
	}
	if envelope.AgentMistake.CookReason != CookMistakeReasonRequestChanges {
		t.Fatalf("cook reason = %q, want %q", envelope.AgentMistake.CookReason, CookMistakeReasonRequestChanges)
	}
	if envelope.AgentMistake.Scope != failure.FailureScopeOrder {
		t.Fatalf("scope = %q, want %q", envelope.AgentMistake.Scope, failure.FailureScopeOrder)
	}
	if envelope.AgentMistake.OrderID != "42" {
		t.Fatalf("agent mistake order_id = %q, want 42", envelope.AgentMistake.OrderID)
	}
	if envelope.AgentMistake.StageIndex == nil || *envelope.AgentMistake.StageIndex != 0 {
		t.Fatalf("agent mistake stage_index = %v, want 0", envelope.AgentMistake.StageIndex)
	}

	events := readNDJSON(t, filepath.Join(l.runtimeDir, "loop-events.ndjson"))
	stageFailed := findEvents(events, LoopEventStageFailed)
	if len(stageFailed) == 0 {
		t.Fatal("expected stage.failed event after request-changes")
	}
	var stagePayload StageFailedPayload
	if err := json.Unmarshal(stageFailed[len(stageFailed)-1].Payload, &stagePayload); err != nil {
		t.Fatalf("parse stage.failed payload: %v", err)
	}
	if stagePayload.AgentMistake == nil {
		t.Fatal("stage.failed payload missing agent_mistake classification")
	}
	if stagePayload.AgentMistake.CookReason != CookMistakeReasonRequestChanges {
		t.Fatalf("stage.failed cook reason = %q, want %q", stagePayload.AgentMistake.CookReason, CookMistakeReasonRequestChanges)
	}
	if stagePayload.Failure == nil {
		t.Fatal("stage.failed payload missing failure classification")
	}
	if stagePayload.Failure.Class != failure.FailureClassAgentMistake {
		t.Fatalf("failure class = %q, want %q", stagePayload.Failure.Class, failure.FailureClassAgentMistake)
	}
}

func TestControlRequestChangesAllowsEmptyFeedback(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	if err := l.controlRequestChanges("42", "   "); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(orders.Orders))
	}
	if orders.Orders[0].Status != OrderStatusFailed {
		t.Fatalf("order status = %q, want %q", orders.Orders[0].Status, OrderStatusFailed)
	}
}

func TestControlRequestChangesNotInPendingReview(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	err := l.controlRequestChanges("nonexistent", "feedback")
	if err == nil {
		t.Fatal("expected error for non-existent pending review item")
	}
	if !strings.Contains(err.Error(), "no pending review") {
		t.Fatalf("error = %q, want 'no pending review'", err.Error())
	}
}

func TestControlRejectKeepsPendingOnWriteFailure(t *testing.T) {
	l := newControlTestLoop(t, &fakeWorktree{}, newMockRuntime())

	// Make the runtime dir unwritable so writeOrdersState fails.
	if err := os.Chmod(l.runtimeDir, 0o444); err != nil {
		t.Fatalf("chmod runtime dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(l.runtimeDir, 0o755) })

	err := l.controlReject("42")
	if err == nil {
		t.Fatal("expected reject to fail when runtime dir is unwritable")
	}
	if _, ok := l.cooks.pendingReview["42"]; !ok {
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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

func TestControlRequeueOrderNotInOrdersFile(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	// Empty orders — order "42" no longer exists.
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
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
	err := l.controlRequeue("42")
	if err == nil {
		t.Fatal("expected error for missing order")
	}
	if !strings.Contains(err.Error(), `order "42" not found`) {
		t.Fatalf("error = %q, want order not found", err.Error())
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    ar,
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})
	l.cooks.pendingReview["42"] = &pendingReviewCook{
		cookIdentity: cookIdentity{orderID: "42", stageIndex: 0, stage: Stage{TaskKey: "execute"}},
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
	if l.cmds.lastAppliedSeq != 2 {
		t.Fatalf("lastAppliedSeq = %d, want 2", l.cmds.lastAppliedSeq)
	}
	if l.cmds.cmdSeqCounter != 2 {
		t.Fatalf("cmdSeqCounter = %d, want 2", l.cmds.cmdSeqCounter)
	}
	if _, ok := l.cmds.processedIDs["c1"]; !ok {
		t.Fatal("c1 should be in processedIDs")
	}
	if _, ok := l.cmds.processedIDs["c2"]; !ok {
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
	if l.cmds.lastAppliedSeq != 1 {
		t.Fatalf("lastAppliedSeq = %d, want 1", l.cmds.lastAppliedSeq)
	}

	// Persist via writeLastAppliedSeq.
	if err := l.writeLastAppliedSeq(); err != nil {
		t.Fatalf("writeLastAppliedSeq: %v", err)
	}

	// Create a new loop and hydrate — should recover the sequence.
	l2 := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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
	if l2.cmds.lastAppliedSeq != 1 {
		t.Fatalf("restored lastAppliedSeq = %d, want 1", l2.cmds.lastAppliedSeq)
	}
	if l2.cmds.cmdSeqCounter != 1 {
		t.Fatalf("restored cmdSeqCounter = %d, want 1", l2.cmds.cmdSeqCounter)
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	// Simulate a restored sequence — commands with seq <= 5 were already applied.
	l.cmds.lastAppliedSeq = 5
	l.cmds.cmdSeqCounter = 5

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
	if l.cmds.lastAppliedSeq != 6 {
		t.Fatalf("lastAppliedSeq = %d, want 6", l.cmds.lastAppliedSeq)
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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

func TestProcessControlLineDerivesDeterministicIDWhenMissing(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        func() time.Time { return now },
		OrdersFile: ordersPath,
	})

	line := `{"action":"pause"}`
	first := l.processControlLine(line)
	second := l.processControlLine(line)

	if first.ID == "" {
		t.Fatal("expected derived command id")
	}
	if first.ID != second.ID {
		t.Fatalf("derived IDs should be deterministic: first=%q second=%q", first.ID, second.ID)
	}
	if !strings.HasPrefix(first.ID, "cmd-auto-") {
		t.Fatalf("derived ID should use cmd-auto prefix: %q", first.ID)
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
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
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

func TestFailedOrderRequiresRequeueBeforeDispatch(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{{
		ID: "42", Title: "test", Status: OrderStatusFailed,
		Stages: []Stage{{TaskKey: "execute", Status: StageStatusFailed}},
	}}}); err != nil {
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
	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	candidates := dispatchableStages(orders, nil, nil, nil)
	if len(candidates) != 0 {
		t.Fatal("failed order should not dispatch before requeue")
	}

	if err := l.controlRequeue("42"); err != nil {
		t.Fatalf("controlRequeue: %v", err)
	}
	orders, err = readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders after requeue: %v", err)
	}
	candidates = dispatchableStages(orders, nil, nil, nil)
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1 after requeue", len(candidates))
	}
}
