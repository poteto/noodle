package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffWorktree(t *testing.T) {
	// Set up a temporary git repo with a main branch and a feature branch.
	// Use os.MkdirTemp instead of t.TempDir() because git background
	// processes on macOS can race with TempDir cleanup, failing the test.
	dir, err := os.MkdirTemp("", "TestDiffWorktree-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Initialize repo with an initial commit on main.
	run("init", "-b", "main")
	run("config", "gc.auto", "0")
	run("config", "core.fsmonitor", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")

	// Create a feature branch and add a change.
	run("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "new.txt")
	run("commit", "-m", "add new.txt")

	// DiffWorktree will fall back to "main" since there's no origin/HEAD.
	result, err := DiffWorktree(dir)
	if err != nil {
		t.Fatalf("DiffWorktree: %v", err)
	}

	if !strings.Contains(result.Diff, "new.txt") {
		t.Errorf("diff should mention new.txt, got:\n%s", result.Diff)
	}
	if !strings.Contains(result.Diff, "+new file") {
		t.Errorf("diff should contain added line, got:\n%s", result.Diff)
	}
	if !strings.Contains(result.Stat, "new.txt") {
		t.Errorf("stat should mention new.txt, got:\n%s", result.Stat)
	}
}

func TestDiffWorktree_NonexistentPath(t *testing.T) {
	_, err := DiffWorktree("/nonexistent/path/does/not/exist")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}
