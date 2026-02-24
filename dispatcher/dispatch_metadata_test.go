package dispatcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteDispatchMetadata(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	createdAt := time.Date(2026, 2, 22, 20, 0, 0, 0, time.UTC)

	err := writeDispatchMetadata(runtimeDir, sessionID, DispatchRequest{
		Provider:     "claude",
		Model:        "claude-sonnet-4-6",
		Runtime:      "sprites",
		Skill:        "debugging",
		WorktreePath: ".worktrees/cook-a",
	}, createdAt)
	if err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	path := filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json")
	var meta dispatchMetadata
	if err := decodeJSONFile(path, &meta); err != nil {
		t.Fatalf("read spawn metadata: %v", err)
	}

	if meta.SessionID != sessionID {
		t.Fatalf("session ID = %q", meta.SessionID)
	}
	if meta.Provider != "claude" {
		t.Fatalf("provider = %q", meta.Provider)
	}
	if meta.Model != "claude-sonnet-4-6" {
		t.Fatalf("model = %q", meta.Model)
	}
	if meta.Runtime != "sprites" {
		t.Fatalf("runtime = %q", meta.Runtime)
	}
	if meta.Skill != "debugging" {
		t.Fatalf("skill = %q", meta.Skill)
	}
	if meta.WorktreePath != ".worktrees/cook-a" {
		t.Fatalf("worktree path = %q", meta.WorktreePath)
	}
	if !meta.CreatedAt.Equal(createdAt) {
		t.Fatalf("created at = %s", meta.CreatedAt)
	}
}

func decodeJSONFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
