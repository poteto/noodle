package startup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestEnsureProjectStructureFreshDirectory(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("EnsureProjectStructure: %v", err)
	}

	// Verify .noodle directory exists
	info, err := os.Stat(filepath.Join(dir, ".noodle"))
	if err != nil {
		t.Fatalf("directory .noodle not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".noodle is not a directory")
	}

	// Verify brain/ is NOT created
	if _, err := os.Stat(filepath.Join(dir, "brain")); err == nil {
		t.Fatal("brain/ should not be created by EnsureProjectStructure")
	}

	// Verify .noodle.toml exists with expected content
	data, err := os.ReadFile(filepath.Join(dir, ".noodle.toml"))
	if err != nil {
		t.Fatalf("file .noodle.toml not created: %v", err)
	}
	if !strings.Contains(string(data), `mode = "supervised"`) {
		t.Fatalf(".noodle.toml missing expected content, got:\n%s", data)
	}

	// Verify generated config is parseable
	parsed, err := config.Parse(data)
	if err != nil {
		t.Fatalf("generated config did not parse: %v", err)
	}
	if parsed.Routing.Defaults.Model != "claude-opus-4-6" {
		t.Fatalf("routing.defaults.model = %q, want claude-opus-4-6", parsed.Routing.Defaults.Model)
	}

	// Verify welcome message was printed
	output := buf.String()
	if !strings.Contains(output, "Noodle initialized") {
		t.Fatalf("expected welcome message on first run, got:\n%s", output)
	}
	if !strings.Contains(output, "INSTALL.md") {
		t.Fatalf("welcome message should mention INSTALL.md, got:\n%s", output)
	}
}

func TestEnsureProjectStructureIdempotent(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	// First run
	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if !strings.Contains(buf.String(), "Noodle initialized") {
		t.Fatal("expected welcome message on first run")
	}

	// Second run
	buf.Reset()
	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("second run: %v", err)
	}

	// No output on second run (nothing created)
	if buf.Len() != 0 {
		t.Fatalf("expected no output on idempotent run, got:\n%s", buf.String())
	}
}

func TestEnsureProjectStructureRecreatesDeletedDirectory(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Delete a required directory
	if err := os.RemoveAll(filepath.Join(dir, ".noodle")); err != nil {
		t.Fatalf("remove .noodle/: %v", err)
	}

	// Re-run should recreate it and show the welcome again
	buf.Reset()
	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(buf.String(), "Noodle initialized") {
		t.Fatalf("expected welcome message after directory deletion, got:\n%s", buf.String())
	}

	info, err := os.Stat(filepath.Join(dir, ".noodle"))
	if err != nil {
		t.Fatalf(".noodle/ not recreated: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".noodle/ is not a directory after recreation")
	}
}
