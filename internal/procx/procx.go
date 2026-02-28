// Package procx provides cross-platform process utilities.
package procx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
)

// IsPIDAlive checks whether a process is alive using kill(pid, 0).
// Returns true on success or EPERM (process exists but owned by another user).
func IsPIDAlive(pid int) bool {
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

// ReadPIDFile reads a JSON file containing a {"pid": N} field and returns the PID.
// Returns an error if the file cannot be read, parsed, or contains an invalid PID.
func ReadPIDFile(path string) (int, error) {
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
