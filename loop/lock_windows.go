//go:build windows

package loop

import "golang.org/x/sys/windows"

func acquireFileLock(fd uintptr) error {
	var overlapped windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(fd),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		^uint32(0),
		^uint32(0),
		&overlapped,
	)
}

func releaseFileLock(fd uintptr) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(
		windows.Handle(fd),
		0,
		^uint32(0),
		^uint32(0),
		&overlapped,
	)
}
