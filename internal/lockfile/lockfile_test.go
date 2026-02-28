package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTryLockSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	lock, err := TryLock(path)
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	defer lock.Close()

	// Lock file should contain our PID.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("lock file is empty")
	}
}

func TestTryLockConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	first, err := TryLock(path)
	if err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	defer first.Close()

	// Second lock on the same file should fail.
	_, err = TryLock(path)
	if err == nil {
		t.Fatal("second TryLock should fail")
	}
	var lockErr *AlreadyLockedError
	if !errors.As(err, &lockErr) {
		t.Fatalf("error type = %T, want *AlreadyLockedError", err)
	}
	if lockErr.HolderPID != os.Getpid() {
		t.Fatalf("HolderPID = %d, want %d", lockErr.HolderPID, os.Getpid())
	}
}

func TestTryLockReleaseThenReacquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")

	first, err := TryLock(path)
	if err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After releasing, another lock should succeed.
	second, err := TryLock(path)
	if err != nil {
		t.Fatalf("second TryLock after release: %v", err)
	}
	second.Close()
}

func TestCloseNilLock(t *testing.T) {
	var lock *Lock
	if err := lock.Close(); err != nil {
		t.Fatalf("Close nil lock: %v", err)
	}
}

func TestAlreadyLockedErrorMessage(t *testing.T) {
	e := &AlreadyLockedError{Path: "/tmp/test.lock", HolderPID: 1234}
	want := "already locked by PID 1234: /tmp/test.lock"
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}

	e2 := &AlreadyLockedError{Path: "/tmp/test.lock"}
	want2 := "already locked: /tmp/test.lock"
	if e2.Error() != want2 {
		t.Fatalf("Error() = %q, want %q", e2.Error(), want2)
	}
}
