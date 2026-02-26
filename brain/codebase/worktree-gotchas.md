# Worktree Gotchas

## CWD inside worktree = permanent shell death

If the shell's cwd is inside a worktree when it gets removed, the shell dies permanently — every subsequent command returns exit code 1 with no recovery.

The Bash tool maintains a **persistent shell** across calls. `cd` in one call drifts the CWD for ALL future calls.

**Prevention:** Never `cd` into a worktree. Use `noodle worktree exec`, subshells `(cd /path && cmd)`, tool flags (`git -C`, `go -C`), or absolute paths.

## `git -C` does NOT protect the shell

`git -C <path>` changes git's working directory, not the shell's. If the shell cwd is inside a worktree, `git -C /project-root worktree remove` still kills the shell.

## Rebase from outside a worktree = fatal error

`git rebase main <branch>` fails with `fatal: '<branch>' is already used by worktree`. Use `git -C <worktree-path> rebase main` instead.

## Concurrent merges cause double-work

When multiple sessions merge to main simultaneously, the second merge's rebase can silently overwrite the first session's edits. The worktree CLI acquires `.worktrees/.merge-lock` before merging. Stale locks are detected via PID.

## Parallel test cleanup can race in ad-hoc git repos

Use `setupTestRepo()` which includes robust cleanup (`git worktree prune`, removing `.git/worktrees`, `.worktrees`). Ad-hoc `t.TempDir()+git init` repos can flake under parallel load.

## Worktree/branch names must be dasherized

`git worktree add ... -b <branch>` rejects names containing `:`. Loop stage names that used `order:stage:task` caused runtime failures like `failed to create worktree: exit status 255`.

Use dasherized names (`order-stage-task`, e.g. `todo-4-0-execute`) for both branch and `.worktrees/` directory safety.

See also [[principles/fix-root-causes]], [[principles/encode-lessons-in-structure]], [[principles/serialize-shared-state-mutations]], [[codebase/adopted-session-reconciliation]]
