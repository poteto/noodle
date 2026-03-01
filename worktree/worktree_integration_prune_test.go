package worktree

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationBranchExplicit(t *testing.T) {
	t.Parallel()
	app := &App{Root: "/unused", IntegrationBranch: "develop"}
	got := app.integrationBranch()
	if got != "develop" {
		t.Errorf("integrationBranch() = %q, want %q", got, "develop")
	}
}

func TestIntegrationBranchFallsBackToMain(t *testing.T) {
	skipWorktreeIntegrationShort(t)

	// No explicit config, no remote HEAD — should fall back to "main".
	dir := setupTestRepo(t)

	app := &App{Root: dir}
	got := app.integrationBranch()
	if got != "main" {
		t.Errorf("integrationBranch() = %q, want %q", got, "main")
	}
}

func TestIntegrationBranchAutoDiscovery(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Set up a fake remote with a non-main HEAD.
	runGitIn(t, dir, "checkout", "-b", "develop")
	runGitIn(t, dir, "checkout", "main")
	// Create a bare clone as the "remote".
	bare := t.TempDir()
	cmd := exec.Command("git", "clone", "--bare", dir, bare)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bare clone failed: %s\n%s", err, out)
	}
	// Point the bare repo HEAD at develop.
	runGitIn(t, bare, "symbolic-ref", "HEAD", "refs/heads/develop")

	// Now set up the local repo to use the bare as origin.
	runGitIn(t, dir, "remote", "add", "origin", bare)
	runGitIn(t, dir, "fetch", "origin")

	app := &App{Root: dir}
	got := app.integrationBranch()
	if got != "develop" {
		t.Errorf("integrationBranch() = %q, want %q (auto-discovered from remote HEAD)", got, "develop")
	}
}

func TestMergeUsesIntegrationBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Rename the default branch to "develop".
	runGitIn(t, dir, "branch", "-M", "develop")

	app := &App{Root: dir, IntegrationBranch: "develop"}

	if err := app.Create("feat-on-develop"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "feat-on-develop")
	writeFile(t, filepath.Join(wtPath, "develop-feat.txt"), "feature on develop")
	runGitIn(t, wtPath, "add", "develop-feat.txt")
	runGitIn(t, wtPath, "commit", "-m", "add feature on develop")

	if err := app.Merge("feat-on-develop"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !fileExists(filepath.Join(dir, "develop-feat.txt")) {
		t.Error("merged file not found on develop branch")
	}
}

func TestCleanupUsesIntegrationBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Rename the default branch to "trunk".
	runGitIn(t, dir, "branch", "-M", "trunk")

	app := &App{Root: dir, IntegrationBranch: "trunk"}

	if err := app.Create("cleanup-trunk"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "cleanup-trunk")
	writeFile(t, filepath.Join(wtPath, "work.txt"), "trunk work")
	runGitIn(t, wtPath, "add", "work.txt")
	runGitIn(t, wtPath, "commit", "-m", "add work on trunk")

	// Cleanup without --force should fail (has unmerged commits).
	err := app.Cleanup("cleanup-trunk")
	if err == nil {
		t.Error("expected error for unmerged commits without --force")
	}

	// Force cleanup should succeed.
	if err := app.Cleanup("cleanup-trunk", CleanupOpts{Force: true}); err != nil {
		t.Fatalf("Cleanup --force failed: %v", err)
	}

	if fileExists(wtPath) {
		t.Error("worktree still exists after force cleanup")
	}
}

func TestMergeRejectsWrongBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Create "develop" branch but stay on main.
	runGitIn(t, dir, "checkout", "-b", "develop")
	runGitIn(t, dir, "checkout", "main")

	// App configured for develop, but repo is on main.
	app := &App{Root: dir, IntegrationBranch: "develop"}

	if err := app.Create("wrong-branch"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err := app.Merge("wrong-branch")
	if err == nil {
		t.Fatal("expected error when not on integration branch")
	}
	if !strings.Contains(err.Error(), "not on develop branch") {
		t.Errorf("error should mention 'not on develop branch', got: %s", err)
	}
}

func TestMergeIntoTargetBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir, Quiet: true}

	// Create a "lead-work" branch to act as the target, and check it out.
	runGitIn(t, dir, "checkout", "-b", "lead-work")

	// Create a worktree (sub-agent work) off of lead-work.
	if err := app.Create("sub-work"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Commit in the sub-work worktree.
	wtPath := WorktreePath(dir, "sub-work")
	writeFile(t, filepath.Join(wtPath, "sub-feature.txt"), "sub-agent work")
	runGitIn(t, wtPath, "add", "sub-feature.txt")
	runGitIn(t, wtPath, "commit", "-m", "add sub-feature")

	// Merge sub-work into lead-work (not main).
	if err := app.Merge("sub-work", MergeOpts{Into: "lead-work"}); err != nil {
		t.Fatalf("Merge --into lead-work failed: %v", err)
	}

	// File should exist on lead-work.
	if !fileExists(filepath.Join(dir, "sub-feature.txt")) {
		t.Error("merged file not found on lead-work branch")
	}

	// Should still be on lead-work.
	branch := gitOutputIn(t, dir, "branch", "--show-current")
	if strings.TrimSpace(branch) != "lead-work" {
		t.Errorf("expected to be on lead-work, got %q", branch)
	}

	// Worktree should be cleaned up.
	if fileExists(wtPath) {
		t.Error("worktree directory still exists after merge")
	}
}

func TestMergeIntoRejectsWrongCurrentBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir, Quiet: true}

	// Stay on main, but try to merge --into a different branch.
	runGitIn(t, dir, "checkout", "-b", "lead-work")
	runGitIn(t, dir, "checkout", "main")

	if err := app.Create("sub-work-2"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Try to merge into lead-work while on main — should fail.
	err := app.Merge("sub-work-2", MergeOpts{Into: "lead-work"})
	if err == nil {
		t.Fatal("expected error when not on target branch")
	}
	if !strings.Contains(err.Error(), "not on lead-work branch") {
		t.Errorf("error should mention 'not on lead-work branch', got: %s", err)
	}
}

func TestListUsesIntegrationBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Rename the default branch to "trunk".
	runGitIn(t, dir, "branch", "-M", "trunk")

	app := &App{Root: dir, IntegrationBranch: "trunk"}

	if err := app.Create("list-trunk"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// List should not error (exercises countUnmergedCommits with custom branch).
	if err := app.List(); err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestPruneRemovesPatchEquivalentWorktrees(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("prune-eq"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	wtPath := WorktreePath(dir, "prune-eq")

	writeFile(t, filepath.Join(wtPath, "equivalent.txt"), "equivalent content")
	runGitIn(t, wtPath, "add", "equivalent.txt")
	runGitIn(t, wtPath, "commit", "-m", "equivalent change")
	branchCommit := gitOutputIn(t, wtPath, "rev-parse", "HEAD")

	// Land equivalent content on main with a different commit hash.
	runGitIn(t, dir, "cherry-pick", branchCommit)

	if err := app.Prune(); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if fileExists(wtPath) {
		t.Fatal("patch-equivalent worktree should be removed by prune")
	}
	branches := gitOutputIn(t, dir, "branch", "--list", "prune-eq")
	if strings.TrimSpace(branches) != "" {
		t.Fatalf("branch prune-eq should be removed, got %q", branches)
	}
}

func TestPruneRemovesPatchEquivalentNoodlePrefixedWorktrees(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}
	name := "prune-prefixed-eq"
	branch := "noodle/" + name

	wtPath := createWorktreeWithBranch(t, dir, name, branch)
	writeFile(t, filepath.Join(wtPath, "equivalent-prefixed.txt"), "equivalent prefixed content")
	runGitIn(t, wtPath, "add", "equivalent-prefixed.txt")
	runGitIn(t, wtPath, "commit", "-m", "equivalent prefixed change")
	branchCommit := gitOutputIn(t, wtPath, "rev-parse", "HEAD")

	// Land equivalent content on main with a different commit hash.
	runGitIn(t, dir, "cherry-pick", branchCommit)

	if err := app.Prune(); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if fileExists(wtPath) {
		t.Fatal("patch-equivalent prefixed worktree should be removed by prune")
	}
	branches := gitOutputIn(t, dir, "branch", "--list", branch)
	if strings.TrimSpace(branches) != "" {
		t.Fatalf("prefixed branch should be removed, got %q", branches)
	}
}

func TestPruneSkipsUnmergedWorktrees(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("prune-keep"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	wtPath := WorktreePath(dir, "prune-keep")

	writeFile(t, filepath.Join(wtPath, "keep.txt"), "keep me")
	runGitIn(t, wtPath, "add", "keep.txt")
	runGitIn(t, wtPath, "commit", "-m", "keep change")

	if err := app.Prune(); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if !fileExists(wtPath) {
		t.Fatal("unmerged worktree should not be removed by prune")
	}
	branches := gitOutputIn(t, dir, "branch", "--list", "prune-keep")
	if strings.TrimSpace(branches) == "" {
		t.Fatal("unmerged branch prune-keep should remain after prune")
	}
}

func TestPruneSkipsUnmergedNestedWorktrees(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}
	name := "feature/prune-keep"

	if err := app.Create(name); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	wtPath := WorktreePath(dir, name)

	writeFile(t, filepath.Join(wtPath, "keep-nested.txt"), "keep nested worktree")
	runGitIn(t, wtPath, "add", "keep-nested.txt")
	runGitIn(t, wtPath, "commit", "-m", "keep nested change")

	if err := app.Prune(); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if !fileExists(wtPath) {
		t.Fatal("unmerged nested worktree should not be removed by prune")
	}
	branches := gitOutputIn(t, dir, "branch", "--list", name)
	if strings.TrimSpace(branches) == "" {
		t.Fatalf("unmerged nested branch %s should remain after prune", name)
	}
}
