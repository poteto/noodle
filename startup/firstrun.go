package startup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureProjectStructure creates the minimum directory and file structure
// needed for noodle to run. Every check is idempotent: if-not-exists, create.
func EnsureProjectStructure(projectDir string, w io.Writer) error {
	dirs := []string{
		filepath.Join(projectDir, "brain"),
		filepath.Join(projectDir, ".noodle"),
	}
	for _, dir := range dirs {
		if err := ensureDir(dir, w); err != nil {
			return err
		}
	}

	files := []struct {
		path    string
		content string
	}{
		{
			path:    filepath.Join(projectDir, "brain", "index.md"),
			content: "# Brain\n",
		},
		{
			path: filepath.Join(projectDir, "brain", "todos.md"),
			content: `# Todos

<!-- next-id: 1 -->
`,
		},
		{
			path:    filepath.Join(projectDir, "brain", "principles.md"),
			content: "# Principles\n",
		},
		{
			path: filepath.Join(projectDir, ".noodle.toml"),
			content: `mode = "supervised"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
`,
		},
	}

	for _, f := range files {
		dir := filepath.Dir(f.path)
		if err := ensureDir(dir, w); err != nil {
			return err
		}
		if err := ensureFile(f.path, f.content, w); err != nil {
			return err
		}
	}

	return nil
}

func ensureDir(path string, w io.Writer) error {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	fmt.Fprintf(w, "created %s/\n", path)
	return nil
}

func ensureFile(path, content string, w io.Writer) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	fmt.Fprintf(w, "created %s\n", path)
	return nil
}
