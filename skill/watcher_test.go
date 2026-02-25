package skill

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcherCallbackOnFileAdd(t *testing.T) {
	dir := t.TempDir()
	var count atomic.Int32

	sw, err := NewSkillWatcher([]string{dir}, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	// Add a file — should trigger callback.
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for count.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("callback not fired within deadline")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if got := count.Load(); got != 1 {
		t.Fatalf("expected callback once, got %d", got)
	}
}

func TestWatcherDebounceCoalesces(t *testing.T) {
	dir := t.TempDir()
	var count atomic.Int32

	sw, err := NewSkillWatcher([]string{dir}, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	// Rapid-fire 5 events within a short window.
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(name, []byte("data"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce to settle.
	time.Sleep(500 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 debounced callback, got %d", got)
	}
}

func TestWatcherChmodOnlyIgnored(t *testing.T) {
	dir := t.TempDir()
	// Pre-create file so we can chmod it.
	filePath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var count atomic.Int32
	sw, err := NewSkillWatcher([]string{dir}, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	// Small delay to let watcher start processing.
	time.Sleep(50 * time.Millisecond)

	// Chmod-only should not trigger callback.
	if err := os.Chmod(filePath, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if got := count.Load(); got != 0 {
		t.Fatalf("expected no callback for chmod-only, got %d", got)
	}
}

func TestWatcherNewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	var count atomic.Int32

	sw, err := NewSkillWatcher([]string{dir}, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	// Create subdirectory — triggers callback + adds watch.
	subDir := filepath.Join(dir, "new-skill")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Wait for debounce from directory creation.
	deadline := time.After(2 * time.Second)
	for count.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("callback not fired for directory creation")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Reset counter, then write a file inside the new subdir.
	count.Store(0)
	if err := os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("# Skill"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	deadline = time.After(2 * time.Second)
	for count.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("callback not fired for file in new subdirectory")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestWatcherRemoveSubdirectory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "skill-to-remove")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var count atomic.Int32
	sw, err := NewSkillWatcher([]string{dir}, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sw.Run(ctx)

	// Small delay to let watcher start.
	time.Sleep(50 * time.Millisecond)

	// Remove the subdirectory — should not panic.
	if err := os.RemoveAll(subDir); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// Wait for debounce.
	deadline := time.After(2 * time.Second)
	for count.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("callback not fired for directory removal")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestWatcherContextCancellation(t *testing.T) {
	dir := t.TempDir()
	sw, err := NewSkillWatcher([]string{dir}, func() {})
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	defer sw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Clean shutdown.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}
}
