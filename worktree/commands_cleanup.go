package worktree

import (
	"fmt"
	"os"
)

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

func (a *App) cleanRemote(branch string) {
	if err := a.git("push", "origin", "--delete", branch).Run(); err != nil {
		a.warnf("WARNING: failed to delete remote branch %s: %v\n", branch, err)
	}
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
