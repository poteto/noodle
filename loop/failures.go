package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (l *Loop) failedPath() string {
	return filepath.Join(l.runtimeDir, "failed.json")
}

func (l *Loop) loadFailedTargets() error {
	path := l.failedPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read failed targets: %w", err)
	}
	var failed map[string]string
	if err := json.Unmarshal(data, &failed); err != nil {
		return fmt.Errorf("parse failed targets: %w", err)
	}
	if l.failedTargets == nil {
		l.failedTargets = map[string]string{}
	}
	for id, reason := range failed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		l.failedTargets[id] = strings.TrimSpace(reason)
	}
	return nil
}

func (l *Loop) markFailed(id string, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if l.failedTargets == nil {
		l.failedTargets = map[string]string{}
	}
	l.failedTargets[id] = strings.TrimSpace(reason)
	return l.writeFailedTargets()
}

func (l *Loop) writeFailedTargets() error {
	path := l.failedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create failed targets directory: %w", err)
	}
	data, err := json.MarshalIndent(l.failedTargets, "", "  ")
	if err != nil {
		return fmt.Errorf("encode failed targets: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write failed targets temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename failed targets file: %w", err)
	}
	return nil
}
