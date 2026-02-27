package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessRuntimeRecoverLivePID(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-live"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write process.json with our own PID (alive).
	procMeta, _ := json.Marshal(map[string]any{
		"pid":        os.Getpid(),
		"session_id": sessionID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), procMeta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}

	// Write meta.json with running status.
	metaJSON, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"status":     "running",
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), metaJSON, 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
	}

	// Write prompt.txt with order tag.
	if err := os.WriteFile(filepath.Join(sessionDir, "prompt.txt"), []byte("[order:fix-bug-42]\nFix the bug."), 0o644); err != nil {
		t.Fatalf("write prompt.txt: %v", err)
	}

	rt := &processRuntime{DispatcherRuntime: NewDispatcherRuntime("process", nil, runtimeDir)}
	recovered, err := rt.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("recovered %d sessions, want 1", len(recovered))
	}
	rs := recovered[0]
	if rs.OrderID != "fix-bug-42" {
		t.Fatalf("order ID = %q, want %q", rs.OrderID, "fix-bug-42")
	}
	if rs.SessionHandle.ID() != sessionID {
		t.Fatalf("session ID = %q, want %q", rs.SessionHandle.ID(), sessionID)
	}
	if rs.Reason != "live PID found" {
		t.Fatalf("reason = %q, want %q", rs.Reason, "live PID found")
	}
}

func TestProcessRuntimeRecoverDeadPIDUpdatesMetaToExited(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-dead"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write process.json with a bogus PID.
	procMeta, _ := json.Marshal(map[string]any{
		"pid":        2147483647,
		"session_id": sessionID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), procMeta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}

	// Write meta.json with running status.
	metaJSON, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"status":     "running",
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), metaJSON, 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
	}

	rt := &processRuntime{DispatcherRuntime: NewDispatcherRuntime("process", nil, runtimeDir)}
	recovered, err := rt.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("recovered %d sessions, want 0", len(recovered))
	}

	// Verify meta.json was updated to "exited".
	data, err := os.ReadFile(filepath.Join(sessionDir, "meta.json"))
	if err != nil {
		t.Fatalf("read meta.json: %v", err)
	}
	if !strings.Contains(string(data), `"status":"exited"`) {
		t.Fatalf("meta.json not updated to exited: %s", string(data))
	}
}

func TestProcessRuntimeRecoverSkipsNoProcessJSON(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-no-process-json"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// meta.json with running status but no process.json.
	metaJSON, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"status":     "running",
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), metaJSON, 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
	}

	rt := &processRuntime{DispatcherRuntime: NewDispatcherRuntime("process", nil, runtimeDir)}
	recovered, err := rt.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("recovered %d sessions, want 0 (no process.json)", len(recovered))
	}

	// meta.json should not have been modified.
	data, err := os.ReadFile(filepath.Join(sessionDir, "meta.json"))
	if err != nil {
		t.Fatalf("read meta.json: %v", err)
	}
	if !strings.Contains(string(data), `"status":"running"`) {
		t.Fatalf("meta.json should still be running: %s", string(data))
	}
}

func TestProcessRuntimeRecoverEmptySessionsDir(t *testing.T) {
	runtimeDir := t.TempDir()
	rt := &processRuntime{DispatcherRuntime: NewDispatcherRuntime("process", nil, runtimeDir)}
	recovered, err := rt.Recover(context.Background())
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("recovered %d sessions, want 0", len(recovered))
	}
}
