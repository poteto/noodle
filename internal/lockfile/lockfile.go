// Package lockfile provides flock-based instance locking.
//
// The lock is kernel-managed: the OS releases it automatically when the
// process exits (including crashes and SIGKILL). The lock file itself
// persists on disk but carries no stale-lock risk because only the
// advisory lock matters, not the file's existence.
package lockfile

import (
	"fmt"
	"os"
	"strconv"
)

// Lock holds an open file descriptor with an exclusive advisory lock.
// Close the lock to release it.
type Lock struct {
	file *os.File
}

// TryLock attempts a non-blocking exclusive lock on path. It creates the
// file if it doesn't exist and writes the current PID into it.
//
// Returns a *Lock on success. If another process holds the lock, returns
// an error wrapping ErrAlreadyLocked. The caller must call Close when done.
func TryLock(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", path, err)
	}

	locked, err := tryFlock(f.Fd())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("flock %s: %w", path, err)
	}
	if !locked {
		// Read the PID of the holder for diagnostics.
		holder := readPID(f)
		f.Close()
		return nil, &AlreadyLockedError{Path: path, HolderPID: holder}
	}

	// Write our PID for diagnostics (other processes can read it).
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	_ = f.Sync()

	return &Lock{file: f}, nil
}

// Close releases the advisory lock and closes the file.
func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// AlreadyLockedError is returned when another process holds the lock.
type AlreadyLockedError struct {
	Path      string
	HolderPID int // 0 if unknown
}

func (e *AlreadyLockedError) Error() string {
	if e.HolderPID > 0 {
		return fmt.Sprintf("already locked by PID %d: %s", e.HolderPID, e.Path)
	}
	return fmt.Sprintf("already locked: %s", e.Path)
}

func readPID(f *os.File) int {
	buf := make([]byte, 32)
	_, _ = f.Seek(0, 0)
	n, _ := f.Read(buf)
	if n == 0 {
		return 0
	}
	// Trim trailing newline.
	s := string(buf[:n])
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	pid, _ := strconv.Atoi(s)
	return pid
}
