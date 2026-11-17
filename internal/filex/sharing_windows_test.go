package filex

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func lockDestination(t *testing.T, destination string) syscall.Handle {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

func TestWriteFileAtomicWaitsForWindowsReader(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "mise.json")
	if err := os.WriteFile(destination, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	handle := lockDestination(t, destination)
	defer func() {
		if handle != syscall.InvalidHandle {
			syscall.CloseHandle(handle)
		}
	}()
	probe := filepath.Join(dir, "probe")
	if err := os.WriteFile(probe, []byte("probe"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(probe, destination); err == nil {
		t.Fatal("test did not establish an actual sharing lock")
	}
	os.Remove(probe)
	done := make(chan error, 1)
	go func() { done <- WriteFileAtomic(destination, []byte("new")) }()
	time.Sleep(75 * time.Millisecond)
	syscall.CloseHandle(handle)
	handle = syscall.InvalidHandle
	if err := <-done; err != nil {
		t.Fatalf("transient sharing violation not recovered: %v", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "new" {
		t.Fatalf("unexpected contents %q, %v", data, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatalf("temporary files leaked: %v", files)
	}
}

func TestWriteFileAtomicPreservesDestinationOnPersistentWindowsLock(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "mise.json")
	if err := os.WriteFile(destination, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	handle := lockDestination(t, destination)
	defer syscall.CloseHandle(handle)
	started := time.Now()
	if err := WriteFileAtomic(destination, []byte("new")); err == nil {
		t.Fatal("persistent lock incorrectly succeeded")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("retry did not remain bounded")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "old" {
		t.Fatalf("old contents lost: %q, %v", data, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatalf("temporary files leaked: %v", files)
	}
}
