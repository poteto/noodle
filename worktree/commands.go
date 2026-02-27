package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Create creates a new worktree and branch, symlinks local settings, and installs deps.
func (a *App) Create(name string) error {
	wtPath := WorktreePath(a.Root, name)

	if a.git("check-ignore", "-q", ".worktrees").Run() != nil {
		f, err := os.OpenFile(
			filepath.Join(a.Root, ".gitignore"),
			os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644,
		)
		if err == nil {
			_, _ = f.WriteString(".worktrees/\n")
			f.Close()
		}
	}

	if _, err := os.Stat(wtPath); err == nil {
		return fmt.Errorf("worktree '%s' already exists at %s", name, wtPath)
	}

	a.info(fmt.Sprintf("Creating worktree at %s...", wtPath))
	if err := a.gitRun("worktree", "add", wtPath, "-b", name); err != nil {
		// Reuse an existing local branch if it already exists but has no worktree path.
		if a.branchExists(name) {
			if retryErr := a.gitRun("worktree", "add", wtPath, name); retryErr == nil {
				goto created
			}
		}
		return fmt.Errorf("failed to create worktree: %w", err)
	}

created:
	settingsPath := filepath.Join(a.Root, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); err == nil {
		wtClaudeDir := filepath.Join(wtPath, ".claude")
		_ = os.MkdirAll(wtClaudeDir, 0o755)
		_ = os.Symlink(settingsPath, filepath.Join(wtClaudeDir, "settings.local.json"))
		a.info("Symlinked .claude/settings.local.json")
	}

	a.installDeps(wtPath)

	a.printf("\nWorktree ready: %s (branch: %s)\n", wtPath, name)
	return nil
}

// Exec runs a command inside the named worktree.
func (a *App) Exec(name string, args []string) error {
	wtPath := WorktreePath(a.Root, name)
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree '%s' does not exist at %s", name, wtPath)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = wtPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout()
	cmd.Stderr = a.stderr()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}

// Merge rebases the worktree branch onto the integration branch, merges, and cleans up.
func (a *App) Merge(name string) error {
	base := a.integrationBranch()
	wtPath := WorktreePath(a.Root, name)
	mergeBranch := a.resolveBranchName(name)
	if err := a.acquireMergeLock(); err != nil {
		return err
	}
	defer a.releaseMergeLock()

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree '%s' does not exist at %s", name, wtPath)
	}
	if err := a.assertCWDSafe(name); err != nil {
		return err
	}

	branch, _ := a.gitOutput("branch", "--show-current")
	if branch != base {
		return fmt.Errorf("not on %s branch (on '%s'). Run: git checkout %s", base, branch, base)
	}

	if err := a.assertRootClean(); err != nil {
		return err
	}

	commits := a.countUnmergedCommits(name)
	if commits == 0 {
		a.info(fmt.Sprintf("No commits on '%s' — nothing to merge", name))
		a.info(fmt.Sprintf("Use '%s cleanup %s' to remove without merging", a.cmdName(), name))
		return nil
	}
	a.info(fmt.Sprintf("%d commit(s) to merge", commits))

	stashed := false
	unstaged := a.git("-C", wtPath, "diff", "--quiet").Run() != nil
	staged := a.git("-C", wtPath, "diff", "--cached", "--quiet").Run() != nil
	if unstaged || staged {
		a.info("Stashing worktree changes before rebase...")
		if a.git("-C", wtPath, "stash", "--include-untracked", "--quiet").Run() == nil {
			stashed = true
		}
	}

	a.info(fmt.Sprintf("Rebasing %s onto %s...", name, base))
	if err := a.git("-C", wtPath, "rebase", base).Run(); err != nil {
		if stashed {
			_ = a.git("-C", wtPath, "rebase", "--abort").Run()
			_ = a.git("-C", wtPath, "stash", "pop", "--quiet").Run()
		}
		if a.hasMergeConflicts(wtPath) {
			return &MergeConflictError{Branch: mergeBranch, Err: err}
		}
		return fmt.Errorf("rebase failed — resolve conflicts manually in %s", wtPath)
	}

	if stashed {
		_ = a.git("-C", wtPath, "stash", "pop", "--quiet").Run()
	}

	a.info(fmt.Sprintf("Merging %s into %s...", mergeBranch, base))
	if err := a.gitRun("merge", mergeBranch); err != nil {
		if a.hasMergeConflicts(a.Root) {
			return &MergeConflictError{Branch: mergeBranch, Err: err}
		}
		return fmt.Errorf("merge failed: %w", err)
	}

	warnings := a.cleanupWorktreeAndBranch(wtPath, mergeBranch, false)

	a.installDeps(a.Root)
	for _, warning := range warnings {
		a.warnf("WARNING: %s\n", warning)
	}

	a.printf("\nDone. %s merged into %s and cleaned up.\n", name, base)
	return nil
}

func (a *App) MergeRemoteBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("remote branch not set")
	}

	base := a.integrationBranch()
	if err := a.acquireMergeLock(); err != nil {
		return err
	}
	defer a.releaseMergeLock()

	current, _ := a.gitOutput("branch", "--show-current")
	if current != base {
		return fmt.Errorf("not on %s branch (on '%s'). Run: git checkout %s", base, current, base)
	}
	if err := a.assertRootClean(); err != nil {
		return err
	}

	a.info(fmt.Sprintf("Fetching remote branch %s...", branch))
	if err := a.gitRun("fetch", "origin", branch); err != nil {
		return fmt.Errorf("fetch remote branch %s: %w", branch, err)
	}

	mergeRef := "origin/" + branch
	a.info(fmt.Sprintf("Merging %s into %s...", mergeRef, base))
	if err := a.gitRun("merge", mergeRef); err != nil {
		if a.hasMergeConflicts(a.Root) {
			return &MergeConflictError{Branch: mergeRef, Err: err}
		}
		return fmt.Errorf("merge remote branch %s: %w", branch, err)
	}

	if err := a.git("push", "origin", "--delete", branch).Run(); err != nil {
		a.warnf("WARNING: failed to delete remote branch %s: %v\n", branch, err)
	}

	a.installDeps(a.Root)
	a.printf("\nDone. %s merged into %s.\n", mergeRef, base)
	return nil
}

// Cleanup removes a worktree without merging. If force is false, it refuses
// when unmerged commits exist.
func (a *App) Cleanup(name string, force bool) error {
	wtPath := a.resolveWorktreePath(name)
	cleanupBranch := a.resolveBranchName(name)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		if !force {
			if err := a.ensureNoUnmergedCommits(name, cleanupBranch); err != nil {
				return err
			}
		}

		a.info("Worktree directory already removed, cleaning up refs...")
		warnings := a.cleanupWorktreeAndBranch("", cleanupBranch, true)
		for _, warning := range warnings {
			a.warnf("WARNING: %s\n", warning)
		}
		a.printf("Done. Cleaned up refs for %s.\n", name)
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if IsCWDInsideWorktree(cwd, wtPath) {
		return fmt.Errorf(
			"shell CWD is inside the worktree (%s).\n  Run first:  cd %s\n  Then retry",
			cwd, a.Root,
		)
	}

	if !force {
		if err := a.ensureNoUnmergedCommits(name, cleanupBranch); err != nil {
			return err
		}
	}

	warnings := a.cleanupWorktreeAndBranch(wtPath, cleanupBranch, true)
	for _, warning := range warnings {
		a.warnf("WARNING: %s\n", warning)
	}
	a.printf("Done. %s removed.\n", name)
	return nil
}

func (a *App) ensureNoUnmergedCommits(name, branch string) error {
	commits := a.countUnmergedCommits(name)
	if commits <= 0 {
		return nil
	}
	base := a.integrationBranch()
	out, _ := a.gitOutput("log", fmt.Sprintf("%s..%s", base, branch), "--oneline")
	a.warnf("WARNING: %s has %d unmerged commit(s):\n%s\n", name, commits, out)
	return fmt.Errorf(
		"use '%s cleanup %s --force' to discard, or '%s merge %s' to keep",
		a.cmdName(), name, a.cmdName(), name,
	)
}

func (a *App) cleanupWorktreeAndBranch(wtPath, branch string, allowForceDelete bool) []string {
	warnings := []string{}
	if wtPath != "" {
		a.info("Removing worktree...")
		if a.git("worktree", "remove", wtPath).Run() != nil {
			if err := a.git("worktree", "remove", "--force", wtPath).Run(); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to remove worktree %s: %v", wtPath, err))
			}
		}
	}

	a.info(fmt.Sprintf("Deleting branch %s...", branch))
	if err := a.git("branch", "-d", branch).Run(); err != nil {
		if allowForceDelete {
			if err := a.git("branch", "-D", branch).Run(); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to delete branch %s: %v", branch, err))
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("failed to delete branch %s: %v", branch, err))
		}
	}

	if err := a.git("worktree", "prune").Run(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to prune worktrees: %v", err))
	}
	return warnings
}

// List shows all worktrees with merge status.
func (a *App) List() error {
	a.printf("Worktrees:\n")
	_ = a.gitRun("worktree", "list")

	base := a.integrationBranch()
	entries, err := a.allWorktreeEntries()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}
	if len(entries) == 0 {
		a.printf("\nNo managed worktrees found.\n")
		return nil
	}

	a.printf("\nBranch status:\n")
	lastSource := ""
	for _, e := range entries {
		if e.Source != lastSource {
			a.printf("  [%s]\n", e.Source)
			lastSource = e.Source
		}
		commits, equivalent, statusErr := a.cherryStatus(e.Name)
		if statusErr != nil {
			a.info(fmt.Sprintf("%s — status unknown (%v)", e.Name, statusErr))
			continue
		}
		if commits == 0 {
			if equivalent > 0 {
				a.info(fmt.Sprintf("%s — patch-equivalent to %s (safe to clean up)", e.Name, base))
			} else {
				a.info(fmt.Sprintf("%s — no commits ahead of %s (safe to clean up)", e.Name, base))
			}
		} else {
			a.info(fmt.Sprintf("%s — %d unmerged commit(s)", e.Name, commits))
		}
	}
	return nil
}

// Prune removes merged worktrees automatically. A worktree is prune-safe when:
// 1) all branch commits are already represented on the integration branch by
// patch equivalence (`git cherry` has no "+" lines), and
// 2) the worktree has no uncommitted changes.
func (a *App) Prune() error {
	base := a.integrationBranch()
	entries, err := a.allWorktreeEntries()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	removed := 0
	skipped := 0
	warnings := []string{}
	for _, e := range entries {
		if !IsRealWorktree(e.Path) {
			skipped++
			warnings = append(warnings, fmt.Sprintf("worktree path missing for %s: %s", e.Name, e.Path))
			continue
		}

		commits, equivalent, statusErr := a.cherryStatus(e.Name)
		if statusErr != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to assess %s against %s: %v", e.Name, base, statusErr))
			continue
		}
		if commits > 0 {
			a.info(fmt.Sprintf("%s — %d unmerged commit(s), skipping", e.Name, commits))
			skipped++
			continue
		}

		clean, cleanErr := a.isWorktreeClean(e.Path)
		if cleanErr != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to read worktree status %s: %v", e.Name, cleanErr))
			continue
		}
		if !clean {
			a.info(fmt.Sprintf("%s — patch-equivalent but dirty, skipping", e.Name))
			skipped++
			continue
		}

		if equivalent > 0 {
			a.info(fmt.Sprintf("%s — patch-equivalent to %s, pruning", e.Name, base))
		} else {
			a.info(fmt.Sprintf("%s — no commits ahead of %s, pruning", e.Name, base))
		}
		if err := a.Cleanup(e.Name, false); err != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to prune %s: %v", e.Name, err))
			continue
		}
		removed++
	}

	if err := a.git("worktree", "prune").Run(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to prune worktree metadata: %v", err))
	}
	for _, warning := range warnings {
		a.warnf("WARNING: %s\n", warning)
	}
	a.printf("\nPrune complete. removed=%d skipped=%d\n", removed, skipped)
	return nil
}
