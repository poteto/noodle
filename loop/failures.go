package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/filex"
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
	if l.cooks.failedTargets == nil {
		l.cooks.failedTargets = map[string]string{}
	}
	for id, reason := range failed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		// Schedule is a system order, not a sticky user failure target.
		// Ignore stale entries so scheduler dispatch is never blocked on restart.
		if strings.EqualFold(id, scheduleOrderID) {
			continue
		}
		l.cooks.failedTargets[id] = strings.TrimSpace(reason)
	}
	return nil
}

func (l *Loop) markFailed(id string, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	// Schedule failures should not become sticky failed targets.
	if strings.EqualFold(id, scheduleOrderID) {
		return nil
	}
	if l.cooks.failedTargets == nil {
		l.cooks.failedTargets = map[string]string{}
	}
	l.cooks.failedTargets[id] = strings.TrimSpace(reason)
	return l.writeFailedTargets()
}

func (l *Loop) writeFailedTargets() error {
	path := l.failedPath()
	data, err := json.MarshalIndent(l.cooks.failedTargets, "", "  ")
	if err != nil {
		return fmt.Errorf("encode failed targets: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("rename failed targets file: %w", err)
	}
	return nil
}
