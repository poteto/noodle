//go:build windows

package lockfile

import "golang.org/x/sys/windows"

// tryFlock attempts a non-blocking exclusive lock. Returns true if acquired.
func tryFlock(fd uintptr) (bool, error) {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(fd),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		^uint32(0),
		^uint32(0),
		&overlapped,
	)
	if err == nil {
		return true, nil
	}
	// ERROR_LOCK_VIOLATION means another process holds the lock.
	if err == windows.ERROR_LOCK_VIOLATION {
		return false, nil
	}
	return false, err
}
