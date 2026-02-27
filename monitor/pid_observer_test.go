package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPidObserverAliveForCurrentProcess(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-pid-alive"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write canonical log so observation has log stats.
	canonicalPath := filepath.Join(sessionDir, "canonical.ndjson")
	if err := os.WriteFile(canonicalPath, []byte("{\"type\":\"init\"}\n"), 0o644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}

	// Write process.json with our own PID (known alive).
	meta, _ := json.Marshal(map[string]any{
		"pid":        os.Getpid(),
		"session_id": sessionID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), meta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}

	observer := NewPidObserver(runtimeDir)
	obs, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !obs.Alive {
		t.Fatal("expected alive for current process PID")
	}
	if obs.LogSize == 0 {
		t.Fatal("expected non-zero log size")
	}
}

func TestPidObserverDeadForBogusProcess(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-pid-dead"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// PID 2147483647 is almost certainly not running.
	meta, _ := json.Marshal(map[string]any{
		"pid":        2147483647,
		"session_id": sessionID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), meta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}

	observer := NewPidObserver(runtimeDir)
	obs, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if obs.Alive {
		t.Fatal("expected dead for bogus PID")
	}
}

func TestPidObserverMissingProcessJsonFallsBack(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-no-process"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// No process.json — observer should return Alive=false without error.
	observer := NewPidObserver(runtimeDir)
	obs, err := observer.Observe(sessionID)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if obs.Alive {
		t.Fatal("expected not alive when process.json is missing")
	}
}

func TestSessionPIDAliveReturnsTrue(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-alive-check"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	meta, _ := json.Marshal(map[string]any{
		"pid":        os.Getpid(),
		"session_id": sessionID,
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), meta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}

	if !SessionPIDAlive(runtimeDir, sessionID) {
		t.Fatal("expected alive for current process")
	}
}

func TestSessionPIDAliveReturnsFalseOnMissing(t *testing.T) {
	runtimeDir := t.TempDir()
	if SessionPIDAlive(runtimeDir, "nonexistent") {
		t.Fatal("expected false for missing session")
	}
}
