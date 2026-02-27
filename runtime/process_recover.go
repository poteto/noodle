package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/poteto/noodle/dispatcher"
)

// processRuntime adds PID-based session recovery to the base DispatcherRuntime.
type processRuntime struct {
	*DispatcherRuntime
}

// NewProcessRuntime adapts a process dispatcher to the Runtime interface with
// PID-based session recovery.
func NewProcessRuntime(d dispatcher.Dispatcher, runtimeDir string, maxConcurrent int) Runtime {
	r := NewDispatcherRuntime("process", d, runtimeDir)
	r.SetMaxConcurrent(maxConcurrent)
	return &processRuntime{DispatcherRuntime: r}
}

func (p *processRuntime) Recover(_ context.Context) ([]RecoveredSession, error) {
	sessionsDir := filepath.Join(p.runtimeDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var recovered []RecoveredSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionDir := filepath.Join(sessionsDir, sessionID)

		pid, err := readRecoverPID(sessionDir)
		if err != nil {
			// No process.json — skip (could be a tmux session).
			continue
		}

		metaPath := filepath.Join(sessionDir, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), `"status":"running"`) {
			continue
		}

		if !recoverPIDAlive(pid) {
			// Dead process — update meta.json status.
			updated := strings.Replace(string(data), `"status":"running"`, `"status":"exited"`, 1)
			_ = os.WriteFile(metaPath, []byte(updated), 0o644)
			continue
		}

		target := ReadSessionTarget(filepath.Join(sessionDir, "prompt.txt"))
		recovered = append(recovered, RecoveredSession{
			OrderID:       target,
			SessionHandle: &recoveredSessionHandle{id: sessionID, status: "running"},
			RuntimeName:   p.name,
			Reason:        "live PID found",
		})
	}

	return recovered, nil
}

// readRecoverPID reads the PID from a session's process.json.
func readRecoverPID(sessionDir string) (int, error) {
	data, err := os.ReadFile(filepath.Join(sessionDir, "process.json"))
	if err != nil {
		return 0, err
	}
	var meta struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0, fmt.Errorf("parse process.json: %w", err)
	}
	if meta.PID <= 0 {
		return 0, fmt.Errorf("invalid PID %d", meta.PID)
	}
	return meta.PID, nil
}

// recoverPIDAlive checks whether a process is alive using kill(pid, 0).
func recoverPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}
