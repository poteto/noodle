//go:build darwin || linux

package lockfile

import "syscall"

// tryFlock attempts a non-blocking exclusive lock. Returns true if acquired.
func tryFlock(fd uintptr) (bool, error) {
	err := syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK {
		return false, nil
	}
	return false, err
}
