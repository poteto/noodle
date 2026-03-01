package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResetRemovesRuntimeState(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")

	// Scaffold some files and subdirectories.
	for _, name := range []string{"orders.json", "mise.json", "noodle.lock"} {
		path := filepath.Join(runtimeDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	sessDir := filepath.Join(runtimeDir, "sessions")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessDir, "s1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
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

	// .noodle/ should exist but be empty.
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		t.Fatalf(".noodle directory should still exist: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf(".noodle should be empty, got %d entries", len(entries))
	}
}

func TestResetToleratesMissingDir(t *testing.T) {
	projectDir := t.TempDir()
	// Don't create .noodle/ at all.

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
		t.Fatalf("runReset should not error when .noodle does not exist: %v", err)
	}

	// Should create the empty directory.
	if _, err := os.Stat(filepath.Join(projectDir, ".noodle")); err != nil {
		t.Errorf(".noodle should exist after reset: %v", err)
	}
}
