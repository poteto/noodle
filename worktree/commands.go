package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		return fmt.Errorf("failed to create worktree: %w", err)
	}

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
	if a.git("-C", wtPath, "rebase", base).Run() != nil {
		if stashed {
			_ = a.git("-C", wtPath, "rebase", "--abort").Run()
			_ = a.git("-C", wtPath, "stash", "pop", "--quiet").Run()
		}
		return fmt.Errorf("rebase failed — resolve conflicts manually in %s", wtPath)
	}

	if stashed {
		_ = a.git("-C", wtPath, "stash", "pop", "--quiet").Run()
	}

	a.info(fmt.Sprintf("Merging %s into %s...", mergeBranch, base))
	if err := a.gitRun("merge", mergeBranch); err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	codexOut := filepath.Join(wtPath, ".codex-output")
	if _, err := os.Stat(codexOut); err == nil {
		a.info("Removing .codex-output/...")
		_ = os.RemoveAll(codexOut)
	}

	a.info("Removing worktree...")
	warnings := []string{}
	if a.git("worktree", "remove", wtPath).Run() != nil {
		if err := a.git("worktree", "remove", "--force", wtPath).Run(); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to remove worktree %s: %v", wtPath, err))
		}
	}

	a.info(fmt.Sprintf("Deleting branch %s...", mergeBranch))
	if err := a.git("branch", "-d", mergeBranch).Run(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to delete branch %s: %v", mergeBranch, err))
	}

	if err := a.git("worktree", "prune").Run(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to prune worktrees: %v", err))
	}

	a.installDeps(a.Root)
	for _, warning := range warnings {
		a.warnf("WARNING: %s\n", warning)
	}

	a.printf("\nDone. %s merged into %s and cleaned up.\n", name, base)
	return nil
}

// Cleanup removes a worktree without merging. If force is false, it refuses
// when unmerged commits exist.
func (a *App) Cleanup(name string, force bool) error {
	wtPath := WorktreePath(a.Root, name)
	cleanupBranch := a.resolveBranchName(name)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		if !force {
			commits := a.countUnmergedCommits(name)
			if commits > 0 {
				base := a.integrationBranch()
				out, _ := a.gitOutput("log", fmt.Sprintf("%s..%s", base, cleanupBranch), "--oneline")
				a.warnf("WARNING: %s has %d unmerged commit(s):\n%s\n", name, commits, out)
				return fmt.Errorf(
					"use '%s cleanup %s --force' to discard, or '%s merge %s' to keep",
					a.cmdName(), name, a.cmdName(), name,
				)
			}
		}

		a.info("Worktree directory already removed, cleaning up refs...")
		warnings := []string{}
		if a.git("branch", "-d", cleanupBranch).Run() != nil {
			if err := a.git("branch", "-D", cleanupBranch).Run(); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to delete branch %s: %v", cleanupBranch, err))
			}
		}
		if err := a.git("worktree", "prune").Run(); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to prune worktrees: %v", err))
		}
		for _, warning := range warnings {
			a.warnf("WARNING: %s\n", warning)
		}
		a.printf("Done. Cleaned up refs for %s.\n", name)
		return nil
	}

	if err := a.assertCWDSafe(name); err != nil {
		return err
	}

	if !force {
		commits := a.countUnmergedCommits(name)
		if commits > 0 {
			base := a.integrationBranch()
			out, _ := a.gitOutput("log", fmt.Sprintf("%s..%s", base, cleanupBranch), "--oneline")
			a.warnf("WARNING: %s has %d unmerged commit(s):\n%s\n", name, commits, out)
			return fmt.Errorf(
				"use '%s cleanup %s --force' to discard, or '%s merge %s' to keep",
				a.cmdName(), name, a.cmdName(), name,
			)
		}
	}

	codexOut := filepath.Join(wtPath, ".codex-output")
	if _, err := os.Stat(codexOut); err == nil {
		_ = os.RemoveAll(codexOut)
	}

	a.info("Removing worktree...")
	warnings := []string{}
	if a.git("worktree", "remove", wtPath).Run() != nil {
		if err := a.git("worktree", "remove", "--force", wtPath).Run(); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to remove worktree %s: %v", wtPath, err))
		}
	}

	a.info(fmt.Sprintf("Deleting branch %s...", cleanupBranch))
	if a.git("branch", "-d", cleanupBranch).Run() != nil {
		if err := a.git("branch", "-D", cleanupBranch).Run(); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to delete branch %s: %v", cleanupBranch, err))
		}
	}

	if err := a.git("worktree", "prune").Run(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to prune worktrees: %v", err))
	}
	for _, warning := range warnings {
		a.warnf("WARNING: %s\n", warning)
	}
	a.printf("Done. %s removed.\n", name)
	return nil
}

// List shows all worktrees with merge status.
func (a *App) List() error {
	a.printf("Worktrees:\n")
	_ = a.gitRun("worktree", "list")

	base := a.integrationBranch()
	a.printf("\nBranch status:\n")
	names, err := a.managedWorktreeNames()
	if err != nil {
		return fmt.Errorf("list managed worktrees: %w", err)
	}
	if len(names) == 0 {
		a.info("No managed worktrees under .worktrees/")
		return nil
	}
	for _, name := range names {
		commits, equivalent, statusErr := a.cherryStatus(name)
		if statusErr != nil {
			a.info(fmt.Sprintf("%s — status unknown (%v)", name, statusErr))
			continue
		}
		if commits == 0 {
			if equivalent > 0 {
				a.info(fmt.Sprintf("%s — patch-equivalent to %s (safe to clean up)", name, base))
			} else {
				a.info(fmt.Sprintf("%s — no commits ahead of %s (safe to clean up)", name, base))
			}
		} else {
			a.info(fmt.Sprintf("%s — %d unmerged commit(s)", name, commits))
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
	names, err := a.managedWorktreeNames()
	if err != nil {
		return fmt.Errorf("list managed worktrees: %w", err)
	}

	removed := 0
	skipped := 0
	warnings := []string{}
	for _, name := range names {
		dir := WorktreePath(a.Root, name)
		if !IsRealWorktree(dir) {
			skipped++
			warnings = append(warnings, fmt.Sprintf("worktree path missing for %s: %s", name, dir))
			continue
		}

		commits, equivalent, statusErr := a.cherryStatus(name)
		if statusErr != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to assess %s against %s: %v", name, base, statusErr))
			continue
		}
		if commits > 0 {
			a.info(fmt.Sprintf("%s — %d unmerged commit(s), skipping", name, commits))
			skipped++
			continue
		}

		clean, cleanErr := a.isWorktreeClean(dir)
		if cleanErr != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to read worktree status %s: %v", name, cleanErr))
			continue
		}
		if !clean {
			a.info(fmt.Sprintf("%s — patch-equivalent but dirty, skipping", name))
			skipped++
			continue
		}

		if equivalent > 0 {
			a.info(fmt.Sprintf("%s — patch-equivalent to %s, pruning", name, base))
		} else {
			a.info(fmt.Sprintf("%s — no commits ahead of %s, pruning", name, base))
		}
		if err := a.Cleanup(name, false); err != nil {
			skipped++
			warnings = append(warnings, fmt.Sprintf("failed to prune %s: %v", name, err))
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
