package spawner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteSpawnMetadata(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	createdAt := time.Date(2026, 2, 22, 20, 0, 0, 0, time.UTC)

	err := writeSpawnMetadata(runtimeDir, sessionID, SpawnRequest{
		Provider:     "claude",
		Model:        "claude-sonnet-4-6",
		Skill:        "debugging",
		WorktreePath: ".worktrees/cook-a",
	}, createdAt)
	if err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	path := filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json")
	var meta spawnMetadata
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
