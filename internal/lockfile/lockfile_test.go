package lockfile

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// TestMain re-execs the test binary as a child process when
// LOCKFILE_TEST_CHILD is set. The child acquires the lock at the path
// given by that env var, prints "locked\n" to stdout, then blocks on
// stdin until the parent closes it (or the process is killed).
func TestMain(m *testing.M) {
	lockPath := os.Getenv("LOCKFILE_TEST_CHILD")
	if lockPath != "" {
		lock, err := TryLock(lockPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "child TryLock: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("locked")
		// Block until parent closes stdin or kills us.
		buf := make([]byte, 1)
		_, _ = os.Stdin.Read(buf)
		lock.Close()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// childCmd builds a command that re-execs the test binary as a lock-
// holding child process. The caller must set cmd.Stdin to a pipe
// (or a file) so the child blocks.
func childCmd(t *testing.T, lockPath string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^$")
	cmd.Env = append(os.Environ(), "LOCKFILE_TEST_CHILD="+lockPath)
	return cmd
}

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

// TestDualProcessExclusion proves that a second OS process cannot acquire
// the lock while the first process holds it. The error must be
// *AlreadyLockedError with the holder's PID.
func TestDualProcessExclusion(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	cmd := childCmd(t, lockPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("child Start: %v", err)
	}
	defer func() {
		stdin.Close()
		_ = cmd.Wait()
	}()

	// Wait for the child to signal it holds the lock.
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != "locked" {
		t.Fatalf("child did not signal readiness; got %q, err %v", scanner.Text(), scanner.Err())
	}

	childPID := cmd.Process.Pid

	// Parent attempts to acquire the same lock — must fail.
	_, err = TryLock(lockPath)
	if err == nil {
		t.Fatal("parent TryLock succeeded while child holds the lock")
	}
	var lockErr *AlreadyLockedError
	if !errors.As(err, &lockErr) {
		t.Fatalf("error type = %T, want *AlreadyLockedError", err)
	}
	if lockErr.HolderPID != childPID {
		t.Errorf("HolderPID = %d, want child PID %d", lockErr.HolderPID, childPID)
	}

	// Verify the error message describes failure state.
	wantMsg := fmt.Sprintf("already locked by PID %d: %s", childPID, lockPath)
	if lockErr.Error() != wantMsg {
		t.Errorf("error message = %q, want %q", lockErr.Error(), wantMsg)
	}
}

// TestStaleLockRecovery proves that flock-based locks are released by the
// kernel when the holding process exits (including crashes). After the
// child exits, the parent must be able to acquire the lock despite the
// lock file still existing on disk with the dead process's PID.
func TestStaleLockRecovery(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	cmd := childCmd(t, lockPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("child Start: %v", err)
	}

	// Wait for the child to signal it holds the lock.
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != "locked" {
		t.Fatalf("child did not signal readiness; got %q, err %v", scanner.Text(), scanner.Err())
	}

	childPID := cmd.Process.Pid

	// Kill the child — simulates a crash. The kernel releases the flock.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill child: %v", err)
	}
	_ = stdin.Close()
	_ = cmd.Wait() // reap zombie

	// The lock file still exists with the dead PID.
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock file missing after child death: %v", err)
	}
	stalePID, _ := strconv.Atoi(string(data[:len(data)-1])) // trim newline
	if stalePID != childPID {
		t.Fatalf("stale PID = %d, want %d", stalePID, childPID)
	}

	// Despite the stale lock file, we can acquire the lock because
	// flock is kernel-managed and was released on process exit.
	lock, err := TryLock(lockPath)
	if err != nil {
		t.Fatalf("TryLock after child death: %v (stale lock not recovered)", err)
	}
	defer lock.Close()

	// Our PID should now be in the file.
	data, err = os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	ourPID, _ := strconv.Atoi(string(data[:len(data)-1]))
	if ourPID != os.Getpid() {
		t.Errorf("lock file PID = %d, want %d", ourPID, os.Getpid())
	}
}
