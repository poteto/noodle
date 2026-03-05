package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/statusfile"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestStampStatusWarningChangeTriggersWrite(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	statusPath := filepath.Join(runtimeDir, "status.json")

	l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        func() time.Time { return time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC) },
		StatusFile: statusPath,
	})

	// First stamp: no warnings.
	if err := l.stampStatus(); err != nil {
		t.Fatalf("first stampStatus: %v", err)
	}
	first := readStatusFile(t, statusPath)
	if len(first.Warnings) != 0 {
		t.Fatalf("first warnings = %v, want empty", first.Warnings)
	}

	// Second stamp with same state should not write (file unchanged).
	info1, _ := os.Stat(statusPath)

	if err := l.stampStatus(); err != nil {
		t.Fatalf("second stampStatus: %v", err)
	}
	info2, _ := os.Stat(statusPath)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatal("expected no write when nothing changed")
	}

	// Third stamp: add warnings — file should be rewritten.
	l.lastMiseWarnings = []string{"adapter flaky"}
	if err := l.stampStatus(); err != nil {
		t.Fatalf("third stampStatus: %v", err)
	}
	third := readStatusFile(t, statusPath)
	if len(third.Warnings) != 1 || third.Warnings[0] != "adapter flaky" {
		t.Fatalf("third warnings = %v, want [adapter flaky]", third.Warnings)
	}
}

func readStatusFile(t *testing.T, path string) statusfile.Status {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read status file: %v", err)
	}
	var s statusfile.Status
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("parse status file: %v", err)
	}
	return s
}
