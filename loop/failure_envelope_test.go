package loop

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/failure"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestCycleClassifiesOrdersNextPromotionFailureAsDegrade(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	ordersNextPath := filepath.Join(runtimeDir, "orders-next.json")
	if err := os.WriteFile(ordersNextPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
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

	envelope := requireLastLoopFailureEnvelope(t, l)
	if envelope.Path != "build.promote_orders_next" {
		t.Fatalf("path = %q, want %q", envelope.Path, "build.promote_orders_next")
	}
	if envelope.Class != CycleFailureClassDegradeContinue {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassDegradeContinue)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityDegrade {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityDegrade)
	}
	if envelope.AgentMistake == nil {
		t.Fatal("agent mistake should be classified for rejected orders-next")
	}
	if envelope.AgentMistake.Owner != failure.FailureOwnerSchedulerAgent {
		t.Fatalf("owner = %q, want %q", envelope.AgentMistake.Owner, failure.FailureOwnerSchedulerAgent)
	}
	if envelope.AgentMistake.SchedulerReason != SchedulerMistakeReasonOrdersNextRejected {
		t.Fatalf("scheduler reason = %q, want %q", envelope.AgentMistake.SchedulerReason, SchedulerMistakeReasonOrdersNextRejected)
	}
	if envelope.AgentMistake.Scope != failure.FailureScopeSystem {
		t.Fatalf("scope = %q, want %q", envelope.AgentMistake.Scope, failure.FailureScopeSystem)
	}

	events := readNDJSON(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	promotions := findEvents(events, LoopEventPromotionFailed)
	if len(promotions) == 0 {
		t.Fatal("expected promotion.failed event")
	}
	var payload PromotionFailedPayload
	if err := json.Unmarshal(promotions[len(promotions)-1].Payload, &payload); err != nil {
		t.Fatalf("parse promotion payload: %v", err)
	}
	if payload.AgentMistake == nil {
		t.Fatal("promotion.failed payload missing agent_mistake classification")
	}
	if payload.AgentMistake.Owner != failure.FailureOwnerSchedulerAgent {
		t.Fatalf("payload owner = %q, want %q", payload.AgentMistake.Owner, failure.FailureOwnerSchedulerAgent)
	}
	if payload.AgentMistake.SchedulerReason != SchedulerMistakeReasonOrdersNextRejected {
		t.Fatalf("payload scheduler reason = %q, want %q", payload.AgentMistake.SchedulerReason, SchedulerMistakeReasonOrdersNextRejected)
	}
	if payload.Failure == nil {
		t.Fatal("promotion.failed payload missing failure classification")
	}
	if payload.Failure.Class != failure.FailureClassAgentMistake {
		t.Fatalf("payload failure class = %q, want %q", payload.Failure.Class, failure.FailureClassAgentMistake)
	}
	if payload.Failure.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("payload failure recoverability = %q, want %q", payload.Failure.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
	if payload.Failure.CycleClass != CycleFailureClassDegradeContinue {
		t.Fatalf("payload cycle class = %q, want %q", payload.Failure.CycleClass, CycleFailureClassDegradeContinue)
	}
}

func TestCycleDoesNotClassifyBackendPromotionFailureAsSchedulerMistake(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders: %v", err)
	}
	ordersNextPath := filepath.Join(runtimeDir, "orders-next.json")
	if err := writeOrdersAtomic(ordersNextPath, OrdersFile{}); err != nil {
		t.Fatalf("write orders-next: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	if err := l.loadOrdersState(); err != nil {
		t.Fatalf("load orders: %v", err)
	}
	if err := os.Remove(ordersPath); err != nil {
		t.Fatalf("remove orders.json: %v", err)
	}
	if err := os.Mkdir(ordersPath, 0o755); err != nil {
		t.Fatalf("replace orders.json with directory: %v", err)
	}

	_, _, err := l.prepareOrdersForCycle(mise.Brief{}, nil, false)
	if err != nil {
		t.Fatalf("prepareOrdersForCycle: %v", err)
	}

	envelope := requireLastLoopFailureEnvelope(t, l)
	if envelope.Path != "build.promote_orders_next" {
		t.Fatalf("path = %q, want %q", envelope.Path, "build.promote_orders_next")
	}
	if envelope.Class != CycleFailureClassDegradeContinue {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassDegradeContinue)
	}
	if envelope.AgentMistake != nil {
		t.Fatalf("agent mistake = %#v, want nil for backend promotion failure", envelope.AgentMistake)
	}

	events := readNDJSON(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	promotions := findEvents(events, LoopEventPromotionFailed)
	if len(promotions) == 0 {
		t.Fatal("expected promotion.failed event")
	}
	var payload PromotionFailedPayload
	if err := json.Unmarshal(promotions[len(promotions)-1].Payload, &payload); err != nil {
		t.Fatalf("parse promotion payload: %v", err)
	}
	if payload.AgentMistake != nil {
		t.Fatalf("payload agent mistake = %#v, want nil for backend promotion failure", payload.AgentMistake)
	}
	if payload.Failure == nil {
		t.Fatal("payload missing failure classification")
	}
	if payload.Failure.Class != failure.FailureClassWarningOnly {
		t.Fatalf("payload failure class = %q, want %q", payload.Failure.Class, failure.FailureClassWarningOnly)
	}
	if payload.Failure.Recoverability != failure.FailureRecoverabilityDegrade {
		t.Fatalf("payload failure recoverability = %q, want %q", payload.Failure.Recoverability, failure.FailureRecoverabilityDegrade)
	}
}

func TestCycleRepairsMissingLifecycleStatusesWithoutCrashing(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	ordersData := []byte(`{
  "orders": [
    {
      "id": "108",
      "title": "statusless order",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-opus-4-6"
        }
      ]
    }
  ]
}`)
	if err := os.WriteFile(ordersPath, ordersData, 0o644); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle should not fail on missing lifecycle statuses: %v", err)
	}

	repaired, err := orderx.ReadOrders(ordersPath)
	if err != nil {
		t.Fatalf("read repaired orders: %v", err)
	}
	if len(repaired.Orders) != 1 {
		t.Fatalf("orders len = %d, want 1", len(repaired.Orders))
	}
	if err := orderx.ValidateOrderStatus(repaired.Orders[0].Status); err != nil {
		t.Fatalf("order status should be valid after repair: %v", err)
	}
	if len(repaired.Orders[0].Stages) != 1 {
		t.Fatalf("stages len = %d, want 1", len(repaired.Orders[0].Stages))
	}
	if err := orderx.ValidateStageStatus(repaired.Orders[0].Stages[0].Status); err != nil {
		t.Fatalf("stage status should be valid after repair: %v", err)
	}
}

func TestCycleClassifiesDispatchFailureAsOrderHard(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{
		Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")},
	}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	rt.dispatchErr = stderrors.New("dispatch failed")

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

	envelope := requireLastLoopFailureEnvelope(t, l)
	if envelope.Path != "cycle.dispatch_terminal" {
		t.Fatalf("path = %q, want %q", envelope.Path, "cycle.dispatch_terminal")
	}
	if envelope.Class != CycleFailureClassOrderHard {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassOrderHard)
	}
	if envelope.OrderClass != OrderFailureClassStageTerminal {
		t.Fatalf("order class = %q, want %q", envelope.OrderClass, OrderFailureClassStageTerminal)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
}

func TestCycleClassifiesFlushFailureAsSystemHard(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{
		Orders: []Order{testOrder("42", "execute", "execute", "claude", "claude-opus-4-6")},
	}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      time.Now,
	})

	brokenRuntimePath := filepath.Join(projectDir, "runtime-file")
	if err := os.WriteFile(brokenRuntimePath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write broken runtime path: %v", err)
	}

	barrierCalls := 0
	l.TestFlushBarrier = func() {
		barrierCalls++
		if barrierCalls == 1 {
			l.runtimeDir = brokenRuntimePath
		}
	}

	err := l.Cycle(context.Background())
	if err == nil {
		t.Fatal("cycle should fail when flushState cannot write pending-review")
	}
	if barrierCalls == 0 {
		t.Fatal("flush barrier was not called")
	}

	envelope := requireLoopFailureEnvelope(t, err)
	if envelope.Path != "persist.flush_state" {
		t.Fatalf("path = %q, want %q", envelope.Path, "persist.flush_state")
	}
	if envelope.Class != CycleFailureClassSystemHard {
		t.Fatalf("class = %q, want %q", envelope.Class, CycleFailureClassSystemHard)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityHard {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityHard)
	}
}

func requireLoopFailureEnvelope(t *testing.T, err error) LoopFailureEnvelope {
	t.Helper()
	var envelope LoopFailureEnvelope
	if !stderrors.As(err, &envelope) {
		t.Fatalf("error = %T (%v), want LoopFailureEnvelope", err, err)
	}
	return envelope
}

func requireLastLoopFailureEnvelope(t *testing.T, l *Loop) LoopFailureEnvelope {
	t.Helper()
	if l.lastLoopFailure == nil {
		t.Fatal("lastLoopFailure should be set")
	}
	return *l.lastLoopFailure
}
