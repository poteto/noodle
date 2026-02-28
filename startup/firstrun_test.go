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

	// Verify directories exist
	for _, sub := range []string{"brain", ".noodle"} {
		info, err := os.Stat(filepath.Join(dir, sub))
		if err != nil {
			t.Fatalf("directory %s not created: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", sub)
		}
	}

	// Verify files exist with expected content
	wantFiles := map[string]string{
		"brain/index.md":       "# Brain",
		"brain/todos.md":       "<!-- next-id: 1 -->",
		"brain/principles.md":  "# Principles",
		".noodle.toml":         `autonomy = "auto"`,
	}
	for relPath, wantContent := range wantFiles {
		data, err := os.ReadFile(filepath.Join(dir, relPath))
		if err != nil {
			t.Fatalf("file %s not created: %v", relPath, err)
		}
		if !strings.Contains(string(data), wantContent) {
			t.Fatalf("%s missing expected content %q, got:\n%s", relPath, wantContent, data)
		}
	}

	// Verify generated config is parseable
	configData, err := os.ReadFile(filepath.Join(dir, ".noodle.toml"))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	parsed, err := config.Parse(configData)
	if err != nil {
		t.Fatalf("generated config did not parse: %v", err)
	}
	if parsed.Routing.Defaults.Model != "claude-opus-4-6" {
		t.Fatalf("routing.defaults.model = %q, want claude-opus-4-6", parsed.Routing.Defaults.Model)
	}

	// Verify something was logged
	if buf.Len() == 0 {
		t.Fatal("expected log output for created files")
	}
}

func TestEnsureProjectStructureIdempotent(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	// First run
	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("first run: %v", err)
	}
	firstOutput := buf.String()

	// Modify a file to prove it's not overwritten
	marker := "# Brain\n\nUser content here.\n"
	if err := os.WriteFile(filepath.Join(dir, "brain", "index.md"), []byte(marker), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
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
	_ = firstOutput

	// Verify marker file was not overwritten
	data, err := os.ReadFile(filepath.Join(dir, "brain", "index.md"))
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if string(data) != marker {
		t.Fatalf("brain/index.md was overwritten: got %q", data)
	}
}

func TestEnsureProjectStructureRecreatesDeletedDirectory(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Delete a required directory
	if err := os.RemoveAll(filepath.Join(dir, "brain")); err != nil {
		t.Fatalf("remove brain/: %v", err)
	}

	// Re-run should recreate it
	buf.Reset()
	if err := EnsureProjectStructure(dir, &buf); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(buf.String(), "brain") {
		t.Fatalf("expected brain directory recreation in output, got:\n%s", buf.String())
	}

	info, err := os.Stat(filepath.Join(dir, "brain"))
	if err != nil {
		t.Fatalf("brain/ not recreated: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("brain/ is not a directory after recreation")
	}
}
