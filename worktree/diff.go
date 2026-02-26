package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiffResult holds the unified diff and diffstat for a worktree branch.
type DiffResult struct {
	Diff string `json:"diff"`
	Stat string `json:"stat"`
}

// DiffWorktree computes the diff between the worktree's branch HEAD and the
// base branch (auto-discovered from origin/HEAD, falling back to "main").
func DiffWorktree(worktreePath string) (DiffResult, error) {
	if _, err := os.Stat(worktreePath); err != nil {
		return DiffResult{}, fmt.Errorf("worktree path not found: %s", worktreePath)
	}

	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("resolve absolute path: %w", err)
	}

	base := discoverBaseBranch(absPath)

	diff, err := gitOutput(absPath, "diff", "--no-ext-diff", "--no-textconv", base+"...HEAD")
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff: %w", err)
	}

	stat, err := gitOutput(absPath, "diff", "--no-ext-diff", "--no-textconv", "--stat", base+"...HEAD")
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --stat: %w", err)
	}

	return DiffResult{Diff: diff, Stat: stat}, nil
}

// discoverBaseBranch returns the default branch name from origin/HEAD,
// falling back to "main" on any error.
func discoverBaseBranch(repoPath string) string {
	ref, err := gitOutput(repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main"
	}
	// ref looks like "refs/remotes/origin/main" — extract branch name.
	if name := strings.TrimPrefix(ref, "refs/remotes/origin/"); name != ref {
		// Guard against ref names starting with "-" which git would
		// interpret as option flags.
		if strings.HasPrefix(name, "-") {
			return "main"
		}
		return name
	}
	return "main"
}

func gitOutput(repoPath string, args ...string) (string, error) {
	full := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", full...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
