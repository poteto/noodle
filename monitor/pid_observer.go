package monitor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/procx"
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

	pidPath := filepath.Join(o.runtimeDir, "sessions", sessionID, "process.json")
	pid, err := procx.ReadPIDFile(pidPath)
	if err != nil {
		// No process.json — can't determine liveness, leave Alive=false.
		return obs, nil
	}

	obs.Alive = procx.IsPIDAlive(pid)
	return obs, nil
}

// SessionPIDAlive returns true if the session's process.json PID is alive.
// Returns false if process.json is missing or the PID is dead.
func SessionPIDAlive(runtimeDir, sessionID string) bool {
	pidPath := filepath.Join(runtimeDir, "sessions", sessionID, "process.json")
	pid, err := procx.ReadPIDFile(pidPath)
	if err != nil {
		return false
	}
	return procx.IsPIDAlive(pid)
}

// TerminateSessionByPID reads the PID from a session's process.json and
// requests graceful termination.
func TerminateSessionByPID(runtimeDir, sessionID string) {
	terminateSessionByPID(runtimeDir, sessionID)
}

// ForceKillSessionByPID reads the PID from a session's process.json and
// forcefully kills the process.
func ForceKillSessionByPID(runtimeDir, sessionID string) {
	forceKillSessionByPID(runtimeDir, sessionID)
}
