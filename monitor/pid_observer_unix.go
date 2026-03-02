//go:build !windows

package monitor

import (
	"path/filepath"
	"syscall"

	"github.com/poteto/noodle/internal/procx"
)

func terminateSessionByPID(runtimeDir, sessionID string) {
	signalSessionByPID(runtimeDir, sessionID, syscall.SIGTERM)
}

func forceKillSessionByPID(runtimeDir, sessionID string) {
	signalSessionByPID(runtimeDir, sessionID, syscall.SIGKILL)
}

func signalSessionByPID(runtimeDir, sessionID string, signal syscall.Signal) {
	pidPath := filepath.Join(runtimeDir, "sessions", sessionID, "process.json")
	pid, err := procx.ReadPIDFile(pidPath)
	if err != nil || !procx.IsPIDAlive(pid) {
		return
	}
	// Try signaling the process group first.
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		_ = syscall.Kill(-pgid, signal)
		return
	}
	// Fall back to signaling just the process.
	_ = syscall.Kill(pid, signal)
}
