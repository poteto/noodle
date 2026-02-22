package spawner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type spawnMetadata struct {
	SessionID    string    `json:"session_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Skill        string    `json:"skill,omitempty"`
	WorktreePath string    `json:"worktree_path,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func writeSpawnMetadata(
	runtimeDir string,
	sessionID string,
	req SpawnRequest,
	createdAt time.Time,
) error {
	runtimeDir = strings.TrimSpace(runtimeDir)
	sessionID = strings.TrimSpace(sessionID)
	if runtimeDir == "" {
		return fmt.Errorf("runtime directory is required")
	}
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	path := filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}

	payload, err := json.Marshal(spawnMetadata{
		SessionID:    sessionID,
		Provider:     strings.TrimSpace(req.Provider),
		Model:        strings.TrimSpace(req.Model),
		Skill:        strings.TrimSpace(req.Skill),
		WorktreePath: strings.TrimSpace(req.WorktreePath),
		CreatedAt:    createdAt.UTC(),
	})
	if err != nil {
		return fmt.Errorf("encode spawn metadata: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write temp spawn metadata: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename spawn metadata: %w", err)
	}
	return nil
}
