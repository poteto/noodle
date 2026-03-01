package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestMarkFailedSkipsScheduleOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
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

	if err := l.markFailed(scheduleOrderID, "cook exited with status failed"); err != nil {
		t.Fatalf("markFailed(schedule): %v", err)
	}
	if _, ok := l.cooks.failedTargets[scheduleOrderID]; ok {
		t.Fatal("schedule should not be recorded as a failed target")
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "failed.json")); !os.IsNotExist(err) {
		t.Fatalf("failed.json should not be created for schedule failures; err=%v", err)
	}
}

func TestLoadFailedTargetsSkipsScheduleOrder(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	failedPath := filepath.Join(runtimeDir, "failed.json")
	content := "{\n  \"schedule\": \"old schedule failure\",\n  \"42\": \"real failure\"\n}\n"
	if err := os.WriteFile(failedPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed.json: %v", err)
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

	if err := l.loadFailedTargets(); err != nil {
		t.Fatalf("loadFailedTargets: %v", err)
	}
	if _, ok := l.cooks.failedTargets[scheduleOrderID]; ok {
		t.Fatal("schedule should be ignored when loading failed targets")
	}
	if got := l.cooks.failedTargets["42"]; got != "real failure" {
		t.Fatalf("failed target 42 = %q, want %q", got, "real failure")
	}
}

func TestClearFailedTargetsForQueuedOrders(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
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
	l.cooks.failedTargets = map[string]string{
		"83":   "previous execution failed",
		"keep": "still failed",
	}
	if err := l.writeFailedTargets(); err != nil {
		t.Fatalf("writeFailedTargets: %v", err)
	}

	orders := OrdersFile{
		Orders: []Order{
			{
				ID:     "83",
				Title:  "retry queued by scheduler",
				Status: OrderStatusActive,
				Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			},
		},
	}
	if err := l.clearFailedTargetsForQueuedOrders(orders); err != nil {
		t.Fatalf("clearFailedTargetsForQueuedOrders: %v", err)
	}

	if _, ok := l.cooks.failedTargets["83"]; ok {
		t.Fatal("failed target 83 should be cleared when order is queued")
	}
	if got := l.cooks.failedTargets["keep"]; got != "still failed" {
		t.Fatalf("failed target keep = %q, want %q", got, "still failed")
	}

	var persisted map[string]string
	data, err := os.ReadFile(filepath.Join(runtimeDir, "failed.json"))
	if err != nil {
		t.Fatalf("read failed.json: %v", err)
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("parse failed.json: %v", err)
	}
	if _, ok := persisted["83"]; ok {
		t.Fatal("persisted failed.json should not include cleared target 83")
	}
	if got := persisted["keep"]; got != "still failed" {
		t.Fatalf("persisted failed target keep = %q, want %q", got, "still failed")
	}

	events := readNDJSON(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	requeued := findEvents(events, event.LoopEventOrderRequeued)
	if len(requeued) != 1 {
		t.Fatalf("order.requeued events = %d, want 1", len(requeued))
	}
	var payload OrderRequeuedPayload
	if err := json.Unmarshal(requeued[0].Payload, &payload); err != nil {
		t.Fatalf("parse order.requeued payload: %v", err)
	}
	if payload.OrderID != "83" {
		t.Fatalf("order.requeued payload order_id = %q, want 83", payload.OrderID)
	}
}
