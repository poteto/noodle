package loop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestStateIncludesBootstrapInFlightInActiveCooks(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	now := time.Date(2026, 3, 3, 5, 35, 0, 0, time.UTC)

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
		Now:      func() time.Time { return now },
		OrdersFile: filepath.Join(runtimeDir, "orders.json"),
	})

	sess := &mockSession{
		id:     "bootstrap-schedule-123",
		status: "running",
		done:   make(chan struct{}),
	}
	l.bootstrapInFlight = &cookHandle{
		cookIdentity: cookIdentity{
			orderID: scheduleOrderID,
			stage: Stage{
				TaskKey: scheduleOrderID,
				Skill:   "bootstrap",
			},
		},
		session:      sess,
		worktreeName: "bootstrap-schedule",
		worktreePath: projectDir,
		startedAt:    now,
	}
	l.publishState()

	state := l.State()
	if len(state.ActiveCooks) != 1 {
		t.Fatalf("active cooks = %d, want 1", len(state.ActiveCooks))
	}
	got := state.ActiveCooks[0]
	if got.SessionID != "bootstrap-schedule-123" {
		t.Fatalf("session id = %q, want bootstrap-schedule-123", got.SessionID)
	}
	if got.Skill != "bootstrap" {
		t.Fatalf("skill = %q, want bootstrap", got.Skill)
	}
	if got.TaskKey != scheduleOrderID {
		t.Fatalf("task_key = %q, want %q", got.TaskKey, scheduleOrderID)
	}
}

