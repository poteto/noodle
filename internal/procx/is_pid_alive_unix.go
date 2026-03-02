//go:build !windows

package procx

import (
	"errors"
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
	return errors.Is(err, syscall.EPERM)
}
