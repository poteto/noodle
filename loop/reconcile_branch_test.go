package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitInRepo(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func TestBranchExistsChecksLocalAndRemoteRefsInOneLookup(t *testing.T) {
	repo := t.TempDir()
	runGitInRepo(t, repo, "init")
	runGitInRepo(t, repo, "config", "user.email", "test@noodle.dev")
	runGitInRepo(t, repo, "config", "user.name", "Noodle Test")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitInRepo(t, repo, "add", "README.md")
	runGitInRepo(t, repo, "commit", "-m", "init")

	runGitInRepo(t, repo, "branch", "feature/local")
	runGitInRepo(t, repo, "update-ref", "refs/remotes/origin/feature/remote", "HEAD")

	if !branchExists(repo, "feature/local") {
		t.Fatal("expected local branch to exist")
	}
	if !branchExists(repo, "feature/remote") {
		t.Fatal("expected remote branch ref to exist")
	}
	if branchExists(repo, "feature/missing") {
		t.Fatal("missing branch should not exist")
	}
	if branchExists(repo, "   ") {
		t.Fatal("blank branch name should not exist")
	}
}
