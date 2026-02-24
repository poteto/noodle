// Command gen-skill generates .agents/skills/noodle/SKILL.md from source metadata.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/generate"
)

func main() {
	content, err := generate.GenerateSkillContent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate noodle skill: %v\n", err)
		os.Exit(1)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repo root: %v\n", err)
		os.Exit(1)
	}

	outPath := filepath.Join(repoRoot, ".agents", "skills", "noodle", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create skill directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write skill: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", outPath)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}
