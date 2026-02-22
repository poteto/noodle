package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateLinkedCheckout resolves and validates a checkout path.
// It enforces that the path exists, is a git checkout, and is NOT a primary
// checkout (".git"). This structurally requires linked worktrees for agent work.
func ValidateLinkedCheckout(path string) (string, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", fmt.Errorf("linked worktree path is required")
	}

	absPath, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat worktree path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("worktree path is not a directory: %s", absPath)
	}

	gitDir, err := absoluteGitDir(absPath)
	if err != nil {
		return "", fmt.Errorf("path is not a git checkout: %s: %w", absPath, err)
	}
	if IsPrimaryCheckout(gitDir) {
		return "", fmt.Errorf("primary checkout is forbidden for agent work: %s", absPath)
	}

	return absPath, nil
}

func absoluteGitDir(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--absolute-git-dir").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --absolute-git-dir: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// MainRepoRoot returns the project root of the main checkout for a linked
// worktree by resolving the git common directory. For a worktree at
// /repo/.worktrees/foo whose git-common-dir is /repo/.git, this returns /repo.
func MainRepoRoot(worktreeDir string) (string, error) {
	out, err := exec.Command("git", "-C", worktreeDir, "rev-parse", "--path-format=absolute", "--git-common-dir").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-common-dir: %s: %w", strings.TrimSpace(string(out)), err)
	}
	commonDir := strings.TrimSpace(string(out)) // e.g. /repo/.git
	return filepath.Dir(commonDir), nil
}

// ProvisionLocalSettings symlinks (or copies as fallback) the project's
// .claude/settings.local.json into a worktree so that spawned sessions
// inherit the same tool permissions as the main checkout.
func ProvisionLocalSettings(projectDir, worktreeDir string) error {
	source := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat source settings.local.json: %w", err)
	}

	targetDir := filepath.Join(worktreeDir, ".claude")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target .claude dir: %w", err)
	}
	target := filepath.Join(targetDir, "settings.local.json")

	if _, err := os.Lstat(target); err == nil {
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("remove existing target settings.local.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat target settings.local.json: %w", err)
	}

	if err := os.Symlink(source, target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrPermission) && !errors.Is(err, errors.ErrUnsupported) {
		return fmt.Errorf("symlink settings.local.json: %w", err)
	}

	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source settings.local.json: %w", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("copy settings.local.json: %w", err)
	}
	return nil
}
