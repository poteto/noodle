package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- test helpers ---

var (
	testRepoFixtureOnce sync.Once
	testRepoFixtureDir  string
	testRepoFixtureErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if testRepoFixtureDir != "" {
		_ = os.RemoveAll(testRepoFixtureDir)
	}
	os.Exit(code)
}

func ensureTestRepoFixture() (string, error) {
	testRepoFixtureOnce.Do(func() {
		dir, err := os.MkdirTemp("", "noodle-worktree-fixture-*")
		if err != nil {
			testRepoFixtureErr = fmt.Errorf("create fixture dir: %w", err)
			return
		}
		testRepoFixtureDir = dir

		if err := runGitFixture(dir, "init", "-q"); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := runGitFixture(dir, "config", "user.email", "test@test.com"); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := runGitFixture(dir, "config", "user.name", "Test"); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := runGitFixture(dir, "config", "gc.auto", "0"); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0o644); err != nil {
			testRepoFixtureErr = fmt.Errorf("write README.md: %w", err)
			return
		}
		if err := runGitFixture(dir, "add", "."); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := runGitFixture(dir, "commit", "-m", "initial commit"); err != nil {
			testRepoFixtureErr = err
			return
		}
		if err := runGitFixture(dir, "branch", "-M", "main"); err != nil {
			testRepoFixtureErr = err
			return
		}

	})

	if testRepoFixtureErr != nil && testRepoFixtureDir != "" {
		_ = os.RemoveAll(testRepoFixtureDir)
		testRepoFixtureDir = ""
	}

	return testRepoFixtureDir, testRepoFixtureErr
}

func runGitFixture(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// setupTestRepo clones a shared fixture repo into a fresh temp directory.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	fixture, err := ensureTestRepoFixture()
	if err != nil {
		t.Fatalf("prepare fixture repo: %v", err)
	}

	root := t.TempDir()
	dir := filepath.Join(root, "repo")
	clone := exec.Command("git", "clone", "-q", fixture, dir)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("git clone fixture failed: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	// Commits in tests run from linked worktrees; set author identity per clone.
	runGitIn(t, dir, "config", "user.email", "test@test.com")
	runGitIn(t, dir, "config", "user.name", "Test")
	runGitIn(t, dir, "remote", "remove", "origin")
	// Disable background git operations that race with t.TempDir() cleanup.
	runGitIn(t, dir, "config", "gc.auto", "0")
	runGitIn(t, dir, "config", "core.fsmonitor", "false")

	// Clean up git worktree refs before t.TempDir() removal.
	// Without this, .git/worktrees/ can make TempDir cleanup fail.
	t.Cleanup(func() {
		exec.Command("git", "-C", dir, "worktree", "prune", "--expire", "now").Run()
		os.RemoveAll(filepath.Join(dir, ".git", "worktrees"))
		os.RemoveAll(filepath.Join(dir, ".worktrees"))
	})

	return dir
}

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
}

func createWorktreeWithBranch(t *testing.T, dir, name, branch string) string {
	t.Helper()
	wtPath := WorktreePath(dir, name)
	runGitIn(t, dir, "worktree", "add", "-b", branch, wtPath)
	return wtPath
}

func gitOutputIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s failed: %s", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func skipWorktreeIntegrationShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping worktree integration test in -short mode")
	}
}

// --- pure helper tests ---

func TestWorktreePath(t *testing.T) {
	t.Parallel()
	got := WorktreePath("/repo", "my-feature")
	want := filepath.Join("/repo", ".worktrees", "my-feature")
	if got != want {
		t.Errorf("WorktreePath = %q, want %q", got, want)
	}
}

func TestIsCWDInsideWorktree(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		cwd    string
		wtPath string
		want   bool
	}{
		{"exact match", "/repo/.worktrees/feat", "/repo/.worktrees/feat", true},
		{"subdirectory", "/repo/.worktrees/feat/src/main.go", "/repo/.worktrees/feat", true},
		{"outside", "/repo/src", "/repo/.worktrees/feat", false},
		{"prefix but not subdir", "/repo/.worktrees/feat-2", "/repo/.worktrees/feat", false},
		{"root of repo", "/repo", "/repo/.worktrees/feat", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsCWDInsideWorktree(tc.cwd, tc.wtPath)
			if got != tc.want {
				t.Errorf("IsCWDInsideWorktree(%q, %q) = %v, want %v",
					tc.cwd, tc.wtPath, got, tc.want)
			}
		})
	}
}

func TestCheckCWDSafe(t *testing.T) {
	t.Parallel()
	t.Run("safe", func(t *testing.T) {
		t.Parallel()
		err := CheckCWDSafe("/repo", "/repo", "feat")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
	t.Run("unsafe", func(t *testing.T) {
		t.Parallel()
		err := CheckCWDSafe("/repo/.worktrees/feat/src", "/repo", "feat")
		if err == nil {
			t.Error("expected error for CWD inside worktree")
		}
		if !strings.Contains(err.Error(), "inside the worktree") {
			t.Errorf("error should mention CWD is inside worktree, got: %s", err)
		}
	})
}

func TestDetectPkgManager(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		lockFile string
		want     string
	}{
		{"pnpm", "pnpm-lock.yaml", "pnpm"},
		{"bun lockb", "bun.lockb", "bun"},
		{"bun lock", "bun.lock", "bun"},
		{"yarn", "yarn.lock", "yarn"},
		{"npm", "package-lock.json", "npm"},
		{"cargo", "Cargo.toml", "cargo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, tc.lockFile), "")
			got := DetectPkgManager(dir)
			if got != tc.want {
				t.Errorf("DetectPkgManager = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("none", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		got := DetectPkgManager(dir)
		if got != "" {
			t.Errorf("DetectPkgManager = %q, want empty", got)
		}
	})
}

func TestInstallArgs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pm      string
		wantBin string
		wantLen int // number of args
	}{
		{"pnpm", "pnpm", 2},
		{"bun", "bun", 2},
		{"yarn", "yarn", 2},
		{"npm", "npm", 1},
		{"cargo", "cargo", 1},
		{"", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.pm, func(t *testing.T) {
			t.Parallel()
			bin, args := InstallArgs(tc.pm)
			if bin != tc.wantBin {
				t.Errorf("InstallArgs(%q) bin = %q, want %q", tc.pm, bin, tc.wantBin)
			}
			if len(args) != tc.wantLen {
				t.Errorf("InstallArgs(%q) args len = %d, want %d", tc.pm, len(args), tc.wantLen)
			}
		})
	}
}

// --- integration tests (real git repos) ---

func TestCreate(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("test-feat"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Worktree directory should exist
	wtPath := WorktreePath(dir, "test-feat")
	if !fileExists(wtPath) {
		t.Error("worktree directory not created")
	}

	// Branch should exist
	branches := gitOutputIn(t, dir, "branch", "--list", "test-feat")
	if !strings.Contains(branches, "test-feat") {
		t.Error("branch not created")
	}

	// .worktrees/ should be gitignored
	if err := exec.Command("git", "-C", dir, "check-ignore", "-q", ".worktrees").Run(); err != nil {
		t.Error(".worktrees/ not gitignored")
	}
}

func TestCreateDuplicate(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("dup"); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := app.Create("dup"); err == nil {
		t.Error("expected error for duplicate worktree")
	}
}

func TestCreateReusesExistingBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	runGitIn(t, dir, "branch", "reuse-branch")

	app := &App{Root: dir}
	if err := app.Create("reuse-branch"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "reuse-branch")
	if !fileExists(wtPath) {
		t.Fatalf("worktree path missing: %s", wtPath)
	}
	if branch := gitOutputIn(t, wtPath, "branch", "--show-current"); branch != "reuse-branch" {
		t.Fatalf("worktree branch = %q, want %q", branch, "reuse-branch")
	}
}

func TestCreateFromBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)

	// Create a branch with an extra commit that main doesn't have.
	runGitIn(t, dir, "checkout", "-b", "base-branch")
	writeFile(t, filepath.Join(dir, "base.txt"), "from base branch")
	runGitIn(t, dir, "add", "base.txt")
	runGitIn(t, dir, "commit", "-m", "base branch commit")
	baseCommit := gitOutputIn(t, dir, "rev-parse", "HEAD")
	runGitIn(t, dir, "checkout", "main")

	app := &App{Root: dir}
	if err := app.Create("from-test", CreateOpts{From: "base-branch"}); err != nil {
		t.Fatalf("Create --from failed: %v", err)
	}

	wtPath := WorktreePath(dir, "from-test")
	if !fileExists(wtPath) {
		t.Fatal("worktree directory not created")
	}

	// The worktree should start at the base-branch commit.
	wtCommit := gitOutputIn(t, wtPath, "rev-parse", "HEAD")
	if wtCommit != baseCommit {
		t.Fatalf("worktree HEAD = %s, want %s (base-branch)", wtCommit, baseCommit)
	}

	// The file from base-branch should be present.
	if !fileExists(filepath.Join(wtPath, "base.txt")) {
		t.Fatal("base.txt not found in worktree — --from did not use correct start point")
	}
}

func TestExec(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("test-exec"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "test-exec")
	writeFile(t, filepath.Join(wtPath, "marker.txt"), "hello")

	if err := app.Exec("test-exec", []string{"cat", "marker.txt"}); err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	err := app.Exec("test-exec", []string{"sh", "-c", "exit 7"})
	if err == nil {
		t.Fatal("expected non-zero exit to return ExitError")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.Code)
	}
}

func TestExecNonExistent(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	err := app.Exec("nonexistent", []string{"echo", "hi"})
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	// Create worktree
	if err := app.Create("test-merge"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Commit a file in the worktree
	wtPath := WorktreePath(dir, "test-merge")
	writeFile(t, filepath.Join(wtPath, "new-file.txt"), "merged content")
	runGitIn(t, wtPath, "add", "new-file.txt")
	runGitIn(t, wtPath, "commit", "-m", "add new-file.txt")

	// Merge back
	if err := app.Merge("test-merge", ""); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// File should now exist in main
	if !fileExists(filepath.Join(dir, "new-file.txt")) {
		t.Error("merged file not found on main")
	}

	// Worktree directory should be gone
	if fileExists(wtPath) {
		t.Error("worktree directory still exists after merge")
	}

	// Branch should be gone
	branches := gitOutputIn(t, dir, "branch", "--list", "test-merge")
	if strings.Contains(branches, "test-merge") {
		t.Error("branch still exists after merge")
	}
}

func TestMergeRemoteBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	runGitIn(t, dir, "init", "--bare", remoteDir)
	runGitIn(t, dir, "remote", "add", "origin", remoteDir)
	runGitIn(t, dir, "config", "merge.ff", "false")
	runGitIn(t, dir, "push", "-u", "origin", "main")

	runGitIn(t, dir, "checkout", "-b", "noodle/remote-merge")
	writeFile(t, filepath.Join(dir, "remote-branch.txt"), "from remote branch\n")
	runGitIn(t, dir, "add", "remote-branch.txt")
	runGitIn(t, dir, "commit", "-m", "add remote-branch.txt")
	runGitIn(t, dir, "push", "-u", "origin", "noodle/remote-merge")
	runGitIn(t, dir, "checkout", "main")

	if err := app.MergeRemoteBranch("noodle/remote-merge"); err != nil {
		t.Fatalf("MergeRemoteBranch failed: %v", err)
	}

	if !fileExists(filepath.Join(dir, "remote-branch.txt")) {
		t.Fatal("merged file not found on main")
	}
	cmd := exec.Command("git", "--git-dir", remoteDir, "branch", "--list", "noodle/remote-merge")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("list remote branches: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("remote branch still exists: %s", strings.TrimSpace(string(out)))
	}
}

func TestMergeRemoteBranchRequiresBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.MergeRemoteBranch(""); err == nil {
		t.Fatal("expected branch validation error")
	}
}

func TestMergeRemoteBranchConflictReturnsTypedError(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	runGitIn(t, dir, "init", "--bare", remoteDir)
	runGitIn(t, dir, "remote", "add", "origin", remoteDir)
	runGitIn(t, dir, "config", "merge.ff", "false")
	runGitIn(t, dir, "push", "-u", "origin", "main")

	runGitIn(t, dir, "checkout", "-b", "noodle/conflict")
	writeFile(t, filepath.Join(dir, "README.md"), "remote change\n")
	runGitIn(t, dir, "add", "README.md")
	runGitIn(t, dir, "commit", "-m", "remote update")
	runGitIn(t, dir, "push", "-u", "origin", "noodle/conflict")
	runGitIn(t, dir, "checkout", "main")

	writeFile(t, filepath.Join(dir, "README.md"), "local change\n")
	runGitIn(t, dir, "add", "README.md")
	runGitIn(t, dir, "commit", "-m", "local update")

	err := app.MergeRemoteBranch("noodle/conflict")
	if err == nil {
		t.Fatal("expected merge conflict error")
	}
	var conflictErr *MergeConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected MergeConflictError, got %T: %v", err, err)
	}
	if conflictErr.Branch != "origin/noodle/conflict" {
		t.Fatalf("conflict branch = %q", conflictErr.Branch)
	}
}

func TestMergeLockAcquiredAndReleased(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("lock-happy-path"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "lock-happy-path")
	writeFile(t, filepath.Join(wtPath, "new-file.txt"), "merged content")
	runGitIn(t, wtPath, "add", "new-file.txt")
	runGitIn(t, wtPath, "commit", "-m", "add new-file.txt")

	if err := app.Merge("lock-happy-path", ""); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after successful merge")
	}
}

func TestMergeLockReleasedOnError(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("lock-failure"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "lock-failure")

	// Commit on branch
	writeFile(t, filepath.Join(wtPath, "README.md"), "branch change\n")
	runGitIn(t, wtPath, "add", "README.md")
	runGitIn(t, wtPath, "commit", "-m", "branch update")

	// Conflicting commit on main to force rebase failure
	writeFile(t, filepath.Join(dir, "README.md"), "main change\n")
	runGitIn(t, dir, "add", "README.md")
	runGitIn(t, dir, "commit", "-m", "main update")

	err := app.Merge("lock-failure", "")
	if err == nil {
		t.Fatal("expected merge to fail due to rebase conflict")
	}
	var conflictErr *MergeConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected MergeConflictError, got: %v", err)
	}
	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after failed merge")
	}
}

func TestReleaseMergeLockOwnership(t *testing.T) {
	dir := t.TempDir()
	app := &App{Root: dir}

	if err := os.MkdirAll(filepath.Join(dir, ".worktrees"), 0755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, app.mergeLockPath(), "99999\n0")
	app.releaseMergeLock()
	if !fileExists(app.mergeLockPath()) {
		t.Fatal("lock should not be removed when owned by another PID")
	}

	writeFile(t, app.mergeLockPath(), fmt.Sprintf("%d\n0", os.Getpid()))
	app.releaseMergeLock()
	if fileExists(app.mergeLockPath()) {
		t.Fatal("lock should be removed when owned by current PID")
	}
}

func TestStaleLockCleanedUp(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir, MergeLockTimeout: 1 * time.Second}

	if err := app.Create("stale-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "stale-lock")
	writeFile(t, filepath.Join(wtPath, "stale.txt"), "stale lock merge")
	runGitIn(t, wtPath, "add", "stale.txt")
	runGitIn(t, wtPath, "commit", "-m", "stale lock commit")

	writeFile(t, app.mergeLockPath(), "99999\n0")

	if err := app.Merge("stale-lock", ""); err != nil {
		t.Fatalf("expected merge to succeed after stale lock cleanup: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after merge completes")
	}
}

func TestActiveLockBlocksMerge(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir, MergeLockTimeout: 350 * time.Millisecond}

	if err := app.Create("active-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "active-lock")
	writeFile(t, filepath.Join(wtPath, "active.txt"), "active lock")
	runGitIn(t, wtPath, "add", "active.txt")
	runGitIn(t, wtPath, "commit", "-m", "active lock commit")

	writeFile(t, app.mergeLockPath(), fmt.Sprintf("%d\n%d", os.Getpid(), time.Now().Unix()))

	err := app.Merge("active-lock", "")
	if err == nil {
		t.Fatal("expected merge to fail when active lock exists")
	}
	if !strings.Contains(err.Error(), "timed out waiting for merge lock") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeLockCorruptContent(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("corrupt-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "corrupt-lock")
	writeFile(t, filepath.Join(wtPath, "corrupt.txt"), "corrupt lock test")
	runGitIn(t, wtPath, "add", "corrupt.txt")
	runGitIn(t, wtPath, "commit", "-m", "corrupt lock commit")

	// Write garbage to the lock file
	writeFile(t, app.mergeLockPath(), "not-a-pid\ngarbage-timestamp")

	if err := app.Merge("corrupt-lock", ""); err != nil {
		t.Fatalf("expected merge to succeed after corrupt lock cleanup: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after merge")
	}
}

func TestMergeLockReleasedOnNoCommits(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("no-commits-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Merge with no commits (no-op path)
	if err := app.Merge("no-commits-lock", ""); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be released even on no-op merge")
	}
}

func TestMergeLockWaitsForRelease(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir, MergeLockTimeout: 1500 * time.Millisecond}

	if err := app.Create("wait-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "wait-lock")
	writeFile(t, filepath.Join(wtPath, "wait.txt"), "wait lock test")
	runGitIn(t, wtPath, "add", "wait.txt")
	runGitIn(t, wtPath, "commit", "-m", "wait lock commit")

	// Create lock with current PID (looks active)
	writeFile(t, app.mergeLockPath(), fmt.Sprintf("%d\n%d", os.Getpid(), time.Now().Unix()))

	mergeDone := make(chan error, 1)
	mergeStarted := make(chan struct{})
	go func() {
		close(mergeStarted)
		mergeDone <- app.Merge("wait-lock", "")
	}()

	<-mergeStarted

	// Merge should be blocked until the lock is removed.
	select {
	case err := <-mergeDone:
		t.Fatalf("expected merge to wait for lock release, got early result: %v", err)
	default:
	}

	if err := os.Remove(app.mergeLockPath()); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove merge lock: %v", err)
	}

	select {
	case err := <-mergeDone:
		if err != nil {
			t.Fatalf("expected merge to succeed after lock release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for merge after lock release")
	}

	if !fileExists(filepath.Join(dir, "wait.txt")) {
		t.Error("merged file not found on main")
	}
}

func TestMergeLockEmptyFile(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("empty-lock"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "empty-lock")
	writeFile(t, filepath.Join(wtPath, "empty.txt"), "empty lock test")
	runGitIn(t, wtPath, "add", "empty.txt")
	runGitIn(t, wtPath, "commit", "-m", "empty lock commit")

	// Write an empty lock file (0 bytes)
	writeFile(t, app.mergeLockPath(), "")

	if err := app.Merge("empty-lock", ""); err != nil {
		t.Fatalf("expected merge to succeed after empty lock cleanup: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after merge")
	}
}

func TestMergeLockZeroPID(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("zero-pid"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "zero-pid")
	writeFile(t, filepath.Join(wtPath, "zero.txt"), "zero pid test")
	runGitIn(t, wtPath, "add", "zero.txt")
	runGitIn(t, wtPath, "commit", "-m", "zero pid commit")

	// Write a lock file with PID 0 (invalid)
	writeFile(t, app.mergeLockPath(), "0\n0")

	if err := app.Merge("zero-pid", ""); err != nil {
		t.Fatalf("expected merge to succeed after zero-PID lock cleanup: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after merge")
	}
}

func TestMergeLockNegativePID(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("neg-pid"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "neg-pid")
	writeFile(t, filepath.Join(wtPath, "neg.txt"), "negative pid test")
	runGitIn(t, wtPath, "add", "neg.txt")
	runGitIn(t, wtPath, "commit", "-m", "neg pid commit")

	// Write a lock file with negative PID
	writeFile(t, app.mergeLockPath(), "-1\n0")

	if err := app.Merge("neg-pid", ""); err != nil {
		t.Fatalf("expected merge to succeed after negative-PID lock cleanup: %v", err)
	}

	if fileExists(app.mergeLockPath()) {
		t.Error("merge lock file should be removed after merge")
	}
}

func TestAcquireMergeLockTimesOut(t *testing.T) {
	// Test acquireMergeLock directly — no git repo needed, just the lock mechanism.
	// The deadline check is at the top of the loop, ensuring ALL paths (stale cleanup,
	// corrupt cleanup, live-process wait) terminate within the timeout.
	dir := t.TempDir()
	app := &App{Root: dir, MergeLockTimeout: 300 * time.Millisecond}

	lockDir := filepath.Join(dir, ".worktrees")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a lock with the current PID (looks active — can't be cleaned up as stale)
	writeFile(t, app.mergeLockPath(), fmt.Sprintf("%d\n%d", os.Getpid(), time.Now().Unix()))

	err := app.acquireMergeLock()

	if err == nil {
		t.Fatal("expected timeout error when lock is held by live process")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	pid, readErr := readMergeLockPID(app.mergeLockPath())
	if readErr != nil {
		t.Fatalf("readMergeLockPID() error = %v", readErr)
	}
	if pid != os.Getpid() {
		t.Fatalf("lock PID = %d, want %d", pid, os.Getpid())
	}
}

func TestMergeNoCommits(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("empty-branch"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Merge with no commits should succeed (no-op)
	if err := app.Merge("empty-branch", ""); err != nil {
		t.Fatalf("Merge with no commits should succeed: %v", err)
	}

	// Worktree should still exist (merge doesn't remove when nothing to merge)
	wtPath := WorktreePath(dir, "empty-branch")
	if !fileExists(wtPath) {
		t.Error("worktree should still exist when there's nothing to merge")
	}
}

func TestMergeNonExistent(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	err := app.Merge("ghost", "")
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("cleanup-test"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "cleanup-test")

	if err := app.Cleanup("cleanup-test", false); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if fileExists(wtPath) {
		t.Error("worktree directory still exists after cleanup")
	}

	branches := gitOutputIn(t, dir, "branch", "--list", "cleanup-test")
	if strings.Contains(branches, "cleanup-test") {
		t.Error("branch still exists after cleanup")
	}
}

func TestCleanupDeletesNoodlePrefixedBranch(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}
	name := "cleanup-prefixed"
	branch := "noodle/" + name

	wtPath := createWorktreeWithBranch(t, dir, name, branch)
	if !fileExists(wtPath) {
		t.Fatalf("expected worktree path to exist: %s", wtPath)
	}

	if err := app.Cleanup(name, false); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if fileExists(wtPath) {
		t.Fatal("worktree directory still exists after cleanup")
	}

	branches := gitOutputIn(t, dir, "branch", "--list", branch)
	if strings.TrimSpace(branches) != "" {
		t.Fatalf("prefixed branch still exists after cleanup: %q", branches)
	}
}

func TestCleanupRefusesUnmergedCommits(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("has-commits"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Make a commit on the branch
	wtPath := WorktreePath(dir, "has-commits")
	writeFile(t, filepath.Join(wtPath, "work.txt"), "important work")
	runGitIn(t, wtPath, "add", "work.txt")
	runGitIn(t, wtPath, "commit", "-m", "add work")

	// Cleanup without --force should fail
	err := app.Cleanup("has-commits", false)
	if err == nil {
		t.Error("expected error for unmerged commits without --force")
	}

	// Worktree should still exist
	if !fileExists(wtPath) {
		t.Error("worktree should still exist after refused cleanup")
	}
}

func TestCleanupForce(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("force-cleanup"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Make a commit
	wtPath := WorktreePath(dir, "force-cleanup")
	writeFile(t, filepath.Join(wtPath, "work.txt"), "will be discarded")
	runGitIn(t, wtPath, "add", "work.txt")
	runGitIn(t, wtPath, "commit", "-m", "doomed commit")

	// Force cleanup should succeed
	if err := app.Cleanup("force-cleanup", true); err != nil {
		t.Fatalf("Cleanup --force failed: %v", err)
	}

	if fileExists(wtPath) {
		t.Error("worktree still exists after force cleanup")
	}
}

func TestCleanupMissingDirectory(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	// Create worktree then manually remove the directory
	if err := app.Create("ghost-dir"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	wtPath := WorktreePath(dir, "ghost-dir")
	os.RemoveAll(wtPath)

	// Cleanup should handle missing directory gracefully
	if err := app.Cleanup("ghost-dir", false); err != nil {
		t.Fatalf("Cleanup of missing dir failed: %v", err)
	}
}

func TestCleanupMissingDirectoryRefusesUnmergedCommits(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("ghost-unmerged"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	wtPath := WorktreePath(dir, "ghost-unmerged")

	writeFile(t, filepath.Join(wtPath, "ghost.txt"), "ghost unmerged")
	runGitIn(t, wtPath, "add", "ghost.txt")
	runGitIn(t, wtPath, "commit", "-m", "ghost unmerged commit")

	os.RemoveAll(wtPath)

	err := app.Cleanup("ghost-unmerged", false)
	if err == nil {
		t.Fatal("expected cleanup to refuse deleting missing worktree refs with unmerged commits")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force guidance in error, got: %v", err)
	}

	branches := gitOutputIn(t, dir, "branch", "--list", "ghost-unmerged")
	if strings.TrimSpace(branches) == "" {
		t.Fatal("ghost-unmerged branch should remain after refused cleanup")
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("list-a"); err != nil {
		t.Fatalf("Create list-a failed: %v", err)
	}
	if err := app.Create("list-b"); err != nil {
		t.Fatalf("Create list-b failed: %v", err)
	}

	// List should not error
	if err := app.List(); err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestMergeWithDirtyWorktree(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	dir := setupTestRepo(t)
	app := &App{Root: dir}

	if err := app.Create("dirty-merge"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	wtPath := WorktreePath(dir, "dirty-merge")

	// Commit a real change
	writeFile(t, filepath.Join(wtPath, "feature.txt"), "feature content")
	runGitIn(t, wtPath, "add", "feature.txt")
	runGitIn(t, wtPath, "commit", "-m", "add feature")

	// Create uncommitted noise (simulates the symlink type-change issue)
	writeFile(t, filepath.Join(wtPath, "noise.txt"), "uncommitted noise")
	runGitIn(t, wtPath, "add", "noise.txt")

	// Merge should handle the dirty worktree via stash
	if err := app.Merge("dirty-merge", ""); err != nil {
		t.Fatalf("Merge with dirty worktree failed: %v", err)
	}

	// The committed file should be on main
	if !fileExists(filepath.Join(dir, "feature.txt")) {
		t.Error("committed file not found on main after merge")
	}
}

// --- integration branch policy tests ---
