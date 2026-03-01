package worktree

import (
	"fmt"
	"os"
	"strings"
)

// Merge rebases the worktree branch onto a target branch, merges, and cleans up.
// When into is non-empty it overrides the integration branch for this merge.
func (a *App) Merge(name, into string) error {
	base := a.integrationBranch()
	if into != "" {
		base = into
	}
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

	a.cleanRemote(branch)

	a.installDeps(a.Root)
	a.printf("\nDone. %s merged into %s.\n", mergeRef, base)
	return nil
}
