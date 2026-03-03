package startup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureProjectStructure creates the minimum directory and file structure
// needed for noodle to run. Every check is idempotent: if-not-exists, create.
// When files are created on first run, a welcome message is printed to w.
func EnsureProjectStructure(projectDir string, w io.Writer) error {
	var created bool

	dirs := []string{
		filepath.Join(projectDir, ".noodle"),
	}
	for _, dir := range dirs {
		made, err := ensureDir(dir)
		if err != nil {
			return err
		}
		created = created || made
	}

	files := []struct {
		path    string
		content string
	}{
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
		if _, err := ensureDir(dir); err != nil {
			return err
		}
		made, err := ensureFile(f.path, f.content)
		if err != nil {
			return err
		}
		created = created || made
	}

	if created {
		fmt.Fprint(w, `Noodle initialized.

Created .noodle/ directory and .noodle.toml config.

Next steps:
  Point your coding agent at INSTALL.md to set up skills and config.
  Open the web UI at the configured address, or run "noodle status" for CLI.
`)
	}

	return nil
}

func ensureDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return false, nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false, fmt.Errorf("create directory %s: %w", path, err)
	}
	return true, nil
}

func ensureFile(path, content string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("create file %s: %w", path, err)
	}
	return true, nil
}
