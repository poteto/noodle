package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestHandleCompletionFailureForwardsToScheduler(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{
		Orders: []Order{{
			ID:     "order-1",
			Status: OrderStatusActive,
			Stages: []Stage{{
				TaskKey: "execute",
				Skill:   "execute",
				Status:  StageStatusActive,
			}},
		}},
	}); err != nil {
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

	schedulerController := &mockController{steerable: true}
	l.cooks.activeCooksByOrder["schedule"] = &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    "schedule",
			stageIndex: 0,
			stage: Stage{
				TaskKey: "schedule",
				Skill:   "schedule",
			},
		},
		session: &steerableSession{
			mockSession: &mockSession{
				id:     "schedule-session",
				status: "running",
				done:   make(chan struct{}),
			},
			ctrl: schedulerController,
		},
	}

	failedCook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    "order-1",
			stageIndex: 0,
			stage: Stage{
				TaskKey: "execute",
				Skill:   "execute",
			},
		},
		session: &mockSession{
			id:     "order-1-session",
			status: "failed",
			done:   make(chan struct{}),
		},
	}

	if err := l.handleCompletion(context.Background(), failedCook, StageResultFailed, "failed"); err != nil {
		t.Fatalf("handleCompletion: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		schedulerController.mu.Lock()
		sendCalls := schedulerController.sendCalls
		lastMessage := schedulerController.lastSentMessage
		schedulerController.mu.Unlock()
		if sendCalls > 0 {
			if !strings.Contains(lastMessage, "[stage_failed]") {
				t.Fatalf("scheduler message = %q, want stage_failed event", lastMessage)
			}
			if !strings.Contains(lastMessage, "order=order-1") {
				t.Fatalf("scheduler message = %q, want order id", lastMessage)
			}
			if !strings.Contains(lastMessage, "cook exited with status failed") {
				t.Fatalf("scheduler message = %q, want failure reason", lastMessage)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	schedulerController.mu.Lock()
	sendCalls := schedulerController.sendCalls
	schedulerController.mu.Unlock()
	if sendCalls == 0 {
		t.Fatal("expected scheduler notification on non-success completion")
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(updated.Orders))
	}
	if updated.Orders[0].Status != OrderStatusFailed {
		t.Fatalf("order status = %q, want %q", updated.Orders[0].Status, OrderStatusFailed)
	}
	if got := updated.Orders[0].Stages[0].Status; got != StageStatusFailed {
		t.Fatalf("stage status = %q, want %q", got, StageStatusFailed)
	}
}
