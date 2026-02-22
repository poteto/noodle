//go:build darwin || linux

package loop

import "syscall"

func acquireFileLock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_EX)
}

func releaseFileLock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
