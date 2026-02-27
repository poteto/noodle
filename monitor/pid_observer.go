package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// PidObserver checks session liveness by reading the PID from process.json
// and probing it with kill(pid, 0). EPERM is treated as alive per
// brain/codebase/unix-process-liveness-eperm.
type PidObserver struct {
	runtimeDir string
}

func NewPidObserver(runtimeDir string) *PidObserver {
	return &PidObserver{runtimeDir: strings.TrimSpace(runtimeDir)}
}

func (o *PidObserver) Observe(sessionID string) (Observation, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Observation{}, fmt.Errorf("session ID is required")
	}
	if o.runtimeDir == "" {
		return Observation{}, fmt.Errorf("runtime directory is required")
	}

	obs, err := canonicalLogObservation(o.runtimeDir, sessionID)
	if err != nil {
		return Observation{}, err
	}

	pid, err := readProcessPID(o.runtimeDir, sessionID)
	if err != nil {
		// No process.json — can't determine liveness, leave Alive=false.
		return obs, nil
	}

	obs.Alive = isPIDAlive(pid)
	return obs, nil
}

// readProcessPID reads the PID from a session's process.json file.
func readProcessPID(runtimeDir, sessionID string) (int, error) {
	path := filepath.Join(runtimeDir, "sessions", sessionID, "process.json")
	data, err := os.ReadFile(path)
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

// isPIDAlive checks whether a process is alive using kill(pid, 0).
// Returns true on success or EPERM (process exists but owned by another user).
func isPIDAlive(pid int) bool {
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

// SessionPIDAlive returns true if the session's process.json PID is alive.
// Returns false if process.json is missing or the PID is dead.
func SessionPIDAlive(runtimeDir, sessionID string) bool {
	pid, err := readProcessPID(runtimeDir, sessionID)
	if err != nil {
		return false
	}
	return isPIDAlive(pid)
}

// KillSessionByPID reads the PID from a session's process.json and sends
// SIGTERM to the process group. Used by loop.Shutdown to kill adopted sessions.
func KillSessionByPID(runtimeDir, sessionID string) {
	pid, err := readProcessPID(runtimeDir, sessionID)
	if err != nil {
		return
	}
	if !isPIDAlive(pid) {
		return
	}
	// Try SIGTERM to the process group first.
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		return
	}
	// Fall back to killing just the process.
	_ = syscall.Kill(pid, syscall.SIGTERM)
}
