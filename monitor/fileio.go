package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionMeta{}, nil
		}
		return SessionMeta{}, fmt.Errorf("read session meta: %w", err)
	}
	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return SessionMeta{}, fmt.Errorf("parse session meta: %w", err)
	}
	return meta, nil
}

func writeSessionMeta(path string, meta SessionMeta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create meta directory: %w", err)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("encode session meta: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "meta-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary meta file: %w", err)
	}
	tmpPath := tmp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary meta file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary meta file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace session meta file: %w", err)
	}
	keepTemp = false
	return nil
}
