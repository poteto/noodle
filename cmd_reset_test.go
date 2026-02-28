package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResetRemovesRuntimeState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")

	// Scaffold files.
	files := []string{
		"orders.json",
		"orders-next.json",
		"mise.json",
		"failed.json",
		"pending-review.json",
		"status.json",
		"tickets.json",
		"control.ndjson",
		"control-ack.ndjson",
		"control.lock",
		"last-applied-seq",
		"loop-events.ndjson",
		"noodle.lock",
	}
	for _, name := range files {
		path := filepath.Join(runtimeDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Scaffold directories.
	dirs := []string{"sessions", "quality"}
	for _, name := range dirs {
		dir := filepath.Join(runtimeDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		// Add a file inside to verify recursive removal.
		if err := os.WriteFile(filepath.Join(dir, "dummy.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write dummy in %s: %v", name, err)
		}
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	app := &App{projectDir: projectDir}
	if err := runReset(app); err != nil {
		t.Fatalf("runReset: %v", err)
	}

	// Verify all files are gone.
	for _, name := range files {
		path := filepath.Join(runtimeDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("file %s still exists after reset", name)
		}
	}

	// Verify all directories are gone.
	for _, name := range dirs {
		path := filepath.Join(runtimeDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("directory %s still exists after reset", name)
		}
	}

	// .noodle/ itself should still exist.
	if _, err := os.Stat(runtimeDir); err != nil {
		t.Errorf(".noodle directory should still exist: %v", err)
	}
}

func TestResetToleratesMissingFiles(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	app := &App{projectDir: projectDir}
	if err := runReset(app); err != nil {
		t.Fatalf("runReset should not error on missing files: %v", err)
	}
}
