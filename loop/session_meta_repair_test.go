package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestStageResultFromSessionMetaStatus(t *testing.T) {
	t.Run("completed-like statuses", func(t *testing.T) {
		for _, status := range []string{"completed", "exited"} {
			got, ok := stageResultFromSessionMetaStatus(status)
			if !ok || got != StageResultCompleted {
				t.Fatalf("status %q => (%q,%v), want (%q,true)", status, got, ok, StageResultCompleted)
			}
		}
	})

	t.Run("failed status", func(t *testing.T) {
		got, ok := stageResultFromSessionMetaStatus("failed")
		if !ok || got != StageResultFailed {
			t.Fatalf("failed => (%q,%v), want (%q,true)", got, ok, StageResultFailed)
		}
	})

	t.Run("cancelled-like statuses", func(t *testing.T) {
		for _, status := range []string{"killed", "cancelled", "canceled", "stopped"} {
			got, ok := stageResultFromSessionMetaStatus(status)
			if !ok || got != StageResultCancelled {
				t.Fatalf("status %q => (%q,%v), want (%q,true)", status, got, ok, StageResultCancelled)
			}
		}
	})

	t.Run("non-terminal status", func(t *testing.T) {
		if _, ok := stageResultFromSessionMetaStatus("running"); ok {
			t.Fatal("running should not be terminal")
		}
	})
}

func TestEnqueueTerminalActiveCompletionsFromMeta(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	orders := OrdersFile{Orders: []Order{
		{
			ID:     "42",
			Status: OrderStatusActive,
			Stages: []Stage{{
				TaskKey:  "execute",
				Skill:    "execute",
				Provider: "claude",
				Model:    "claude-opus-4-6",
				Status:   StageStatusActive,
			}},
		},
	}}
	ordersPath := writeTestOrders(t, runtimeDir, orders)

	sessionID := "42-0-execute-session"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), []byte(`{"status":"exited"}`), 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
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

	done := make(chan struct{})
	sess := &mockSession{id: sessionID, status: "running", done: done}
	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    "42",
			stageIndex: 0,
			stage:      orders.Orders[0].Stages[0],
		},
		orderStatus:  OrderStatusActive,
		session:      sess,
		worktreeName: "42-0-execute",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42-0-execute"),
		generation:   7,
	}
	l.cooks.activeCooksByOrder["42"] = cook

	if err := l.enqueueTerminalActiveCompletions(context.Background()); err != nil {
		t.Fatalf("enqueueTerminalActiveCompletions: %v", err)
	}

	if got := sess.Status(); got != "killed" {
		t.Fatalf("session status = %q, want killed after repair-triggered close", got)
	}

	select {
	case result := <-l.completionBuf.completions:
		if result.OrderID != "42" {
			t.Fatalf("result order = %q, want 42", result.OrderID)
		}
		if result.Status != StageResultCompleted {
			t.Fatalf("result status = %q, want %q", result.Status, StageResultCompleted)
		}
		if result.Generation != 7 {
			t.Fatalf("result generation = %d, want 7", result.Generation)
		}
	default:
		t.Fatal("expected completion to be enqueued")
	}
}

func TestEnqueueTerminalBootstrapCompletionFromMeta(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	sessionID := "bootstrap-schedule-test"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), []byte(`{"status":"exited"}`), 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
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

	done := make(chan struct{})
	sess := &mockSession{id: sessionID, status: "running", done: done}
	l.bootstrapInFlight = &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    scheduleOrderID,
			stageIndex: 0,
			stage: Stage{
				TaskKey: scheduleOrderID,
				Skill:   "bootstrap",
			},
		},
		orderStatus:  OrderStatusActive,
		session:      sess,
		worktreeName: "bootstrap-schedule",
		worktreePath: projectDir,
		generation:   11,
	}

	if err := l.enqueueTerminalActiveCompletions(context.Background()); err != nil {
		t.Fatalf("enqueueTerminalActiveCompletions: %v", err)
	}

	if got := sess.Status(); got != "killed" {
		t.Fatalf("session status = %q, want killed after repair-triggered close", got)
	}

	select {
	case result := <-l.completionBuf.completions:
		if result.Status != StageResultCompleted {
			t.Fatalf("result status = %q, want %q", result.Status, StageResultCompleted)
		}
		if !result.IsBootstrap {
			t.Fatal("expected bootstrap completion result")
		}
		if result.Generation != 11 {
			t.Fatalf("result generation = %d, want 11", result.Generation)
		}
	default:
		t.Fatal("expected bootstrap completion to be enqueued")
	}
}
