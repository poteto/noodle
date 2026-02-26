package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
)

func TestParkPendingReviewWritesFile(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, Queue{}); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	cook := &activeCook{
		queueItem: QueueItem{ID: "42", TaskKey: "execute", Title: "Implement fix", Skill: "execute"},
		session:   &adoptedSession{id: "sess-42", status: "completed"},
		worktreeName: "42",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42"),
	}
	if err := l.parkPendingReview(cook); err != nil {
		t.Fatalf("park pending review: %v", err)
	}

	items, err := ReadPendingReview(runtimeDir)
	if err != nil {
		t.Fatalf("read pending review file: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("pending review items = %d, want 1", len(items))
	}
	if items[0].ID != "42" {
		t.Fatalf("item id = %q, want 42", items[0].ID)
	}
	if items[0].SessionID != "sess-42" {
		t.Fatalf("session id = %q, want sess-42", items[0].SessionID)
	}
}

func TestLoadPendingReviewHydratesLoopState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	payload := `{
  "items": [
    {
      "id": "42",
      "task_key": "execute",
      "title": "Implement fix",
      "worktree_name": "42",
      "worktree_path": "` + filepath.Join(projectDir, ".worktrees", "42") + `",
      "session_id": "sess-42"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(runtimeDir, "pending-review.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write pending review: %v", err)
	}

	queuePath := filepath.Join(runtimeDir, "queue.json")
	if err := writeQueueAtomic(queuePath, Queue{}); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})

	if err := l.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(l.pendingReview) != 1 {
		t.Fatalf("pendingReview size = %d, want 1", len(l.pendingReview))
	}
	pending, ok := l.pendingReview["42"]
	if !ok {
		t.Fatal("expected item 42 in pending review")
	}
	if pending.sessionID != "sess-42" {
		t.Fatalf("session id = %q, want sess-42", pending.sessionID)
	}
}

func TestPlanCycleSpawnsSkipsPendingReviewTargets(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")
	queue := Queue{Items: []QueueItem{{ID: "42"}, {ID: "43"}}}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Concurrency.MaxCooks = 2
	l := New(projectDir, "noodle", cfg, Dependencies{
		Dispatcher: &fakeDispatcher{},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		QueueFile:  queuePath,
	})
	l.pendingReview["42"] = &pendingReviewCook{queueItem: QueueItem{ID: "42"}}

	plan := l.planCycleSpawns(queue, mise.Brief{}, l.config.Concurrency.MaxCooks)
	if len(plan) != 1 || plan[0].ID != "43" {
		t.Fatalf("spawn plan = %#v, want only 43", plan)
	}
}
