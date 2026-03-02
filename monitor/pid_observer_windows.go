//go:build windows

package monitor

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/internal/procx"
)

func terminateSessionByPID(runtimeDir, sessionID string) {
	pid, ok := sessionPID(runtimeDir, sessionID)
	if !ok {
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		_ = process.Kill()
	}
}

func forceKillSessionByPID(runtimeDir, sessionID string) {
	pid, ok := sessionPID(runtimeDir, sessionID)
	if !ok {
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Kill()
}

func sessionPID(runtimeDir, sessionID string) (int, bool) {
	pidPath := filepath.Join(runtimeDir, "sessions", sessionID, "process.json")
	pid, err := procx.ReadPIDFile(pidPath)
	if err != nil || !procx.IsPIDAlive(pid) {
		return 0, false
	}
	return pid, true
}
