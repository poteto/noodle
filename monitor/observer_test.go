package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTmuxObserverDetectsDeadPaneAndLogStats(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	canonicalPath := filepath.Join(sessionPath, "canonical.ndjson")
	content := []byte("{\"type\":\"init\",\"timestamp\":\"2026-02-22T15:00:00Z\"}\n")
	if err := os.WriteFile(canonicalPath, content, 0o644); err != nil {
		t.Fatalf("write canonical file: %v", err)
	}

	observer := NewTmuxObserver(runtimeDir)
	observer.run = func(name string, args ...string) error {
		if name != "tmux" {
			return fmt.Errorf("unexpected command %q", name)
		}
		return fmt.Errorf("session missing")
	}

	observation, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe session: %v", err)
	}
	if observation.Alive {
		t.Fatal("expected dead pane")
	}
	if observation.LogSize == 0 {
		t.Fatal("expected non-zero log size")
	}
	if observation.LogMTime.IsZero() {
		t.Fatal("expected log mtime")
	}
	if observation.LogMTime.After(time.Now().Add(1 * time.Second)) {
		t.Fatalf("unexpected mtime in future: %v", observation.LogMTime)
	}
}
