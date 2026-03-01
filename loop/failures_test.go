package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
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
