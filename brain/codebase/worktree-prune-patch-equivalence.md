# Worktree Prune Uses Patch Equivalence

- `noodle worktree prune` auto-cleans only worktrees that are safe by content, not just by commit ancestry.
- Safety rule is based on `git cherry <base> <branch>`:
  - no `+` lines means every branch commit is already represented on the integration branch (`main` by default, or configured integration branch),
  - `-` lines are patch-equivalent commits already landed with different hashes (for example via cherry-pick or rebase).
- Prune also requires a clean worktree (`git status --porcelain` empty). Patch-equivalent but dirty worktrees are skipped.
- Stale directories in `.worktrees/` without git metadata are removed as filesystem cleanup.

See also [[codebase/worktree-gotchas]], [[codebase/worktree-skill-entrypoint]]
