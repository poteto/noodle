package loop

import (
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
	queuePath := filepath.Join(runtimeDir, "queue.json")
	ordersPath := filepath.Join(runtimeDir, "orders.json")
	if err := writeQueueAtomic(queuePath, Queue{Items: []QueueItem{{
		ID: "42", TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6",
	}}}); err != nil {
		t.Fatalf("write queue: %v", err)
	}
	// Write orders.json for the new control paths.
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
		QueueFile:  queuePath,
		OrdersFile: ordersPath,
	})
	l.pendingReview["42"] = &pendingReviewCook{
		orderID:      "42",
		stageIndex:   0,
		stage:        Stage{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "claude-opus-4-6"},
		worktreeName: "42:0:execute",
		worktreePath: filepath.Join(projectDir, ".worktrees", "42:0:execute"),
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

func TestControlRequestChangesSpawnsInSameWorktree(t *testing.T) {
	sp := &fakeDispatcher{}
	l := newControlTestLoop(t, &fakeWorktree{}, sp)

	if err := l.controlRequestChanges("42", "Add missing tests"); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	if len(sp.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1", len(sp.calls))
	}
	if got := sp.calls[0].WorktreePath; got != l.worktreePath("42:0:execute") {
		t.Fatalf("worktree path = %q, want %q", got, l.worktreePath("42:0:execute"))
	}
	if !strings.Contains(sp.calls[0].Prompt, "Previous work needs changes. Feedback: Add missing tests") {
		t.Fatalf("prompt missing feedback context: %q", sp.calls[0].Prompt)
	}
	if _, ok := l.pendingReview["42"]; ok {
		t.Fatal("pending review item should be removed after successful request-changes")
	}
	if _, ok := l.activeByTarget["42"]; !ok {
		t.Fatal("expected item to be active after request-changes spawn")
	}
}

func TestControlRequestChangesAllowsEmptyFeedback(t *testing.T) {
	sp := &fakeDispatcher{}
	l := newControlTestLoop(t, &fakeWorktree{}, sp)

	if err := l.controlRequestChanges("42", "   "); err != nil {
		t.Fatalf("controlRequestChanges: %v", err)
	}
	if !strings.Contains(sp.calls[0].Prompt, "Previous work needs changes.") {
		t.Fatalf("prompt missing base request-changes instruction: %q", sp.calls[0].Prompt)
	}
}

func TestControlRequestChangesKeepsPendingOnDispatchFailure(t *testing.T) {
	sp := &fakeDispatcher{dispatchErr: errors.New("dispatch failed")}
	l := newControlTestLoop(t, &fakeWorktree{}, sp)

	err := l.controlRequestChanges("42", "Try again")
	if err == nil {
		t.Fatal("expected dispatch failure")
	}
	if _, ok := l.pendingReview["42"]; !ok {
		t.Fatal("pending review item should remain when request-changes dispatch fails")
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
