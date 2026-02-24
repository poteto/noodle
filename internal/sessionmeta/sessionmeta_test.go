package sessionmeta

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadSessionAndReadAll(t *testing.T) {
	runtimeDir := t.TempDir()
	sessions := filepath.Join(runtimeDir, "sessions")
	if err := os.MkdirAll(filepath.Join(sessions, "a"), 0o755); err != nil {
		t.Fatalf("mkdir a: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sessions, "b"), 0o755); err != nil {
		t.Fatalf("mkdir b: %v", err)
	}

	tA := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	tB := time.Date(2026, 2, 20, 11, 0, 0, 0, time.UTC)

	metaA := `{"session_id":"a","status":"RUNNING","runtime":" sprites ","updated_at":"` + tA.Format(time.RFC3339) + `"}`
	metaB := `{"session_id":"b","status":"exited","updated_at":"` + tB.Format(time.RFC3339) + `"}`

	if err := os.WriteFile(filepath.Join(sessions, "a", "meta.json"), []byte(metaA), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessions, "b", "meta.json"), []byte(metaB), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	meta, err := ReadSession(runtimeDir, "a")
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if meta.Status != "running" {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.Runtime != "sprites" {
		t.Fatalf("runtime = %q", meta.Runtime)
	}

	all, err := ReadAll(runtimeDir)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("all len = %d", len(all))
	}
	if all[0].SessionID != "b" {
		t.Fatalf("expected newest first, got %q", all[0].SessionID)
	}
}
