package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/poteto/noodle/internal/jsonx"
)

func listSessionIDs(runtimeDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(runtimeDir, "sessions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	sessionIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionIDs = append(sessionIDs, entry.Name())
	}
	sort.Strings(sessionIDs)
	return sessionIDs, nil
}

func sessionMetaPath(runtimeDir, sessionID string) string {
	return filepath.Join(runtimeDir, "sessions", sessionID, "meta.json")
}

func readSessionMeta(path string) (SessionMeta, error) {
	return jsonx.ReadJSON[SessionMeta](path)
}

func writeSessionMeta(path string, meta SessionMeta) error {
	return jsonx.WriteJSON(path, meta)
}
