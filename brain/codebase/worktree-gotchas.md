# Worktree Gotchas

## CWD inside worktree = permanent shell death

If the shell's cwd is inside a worktree when it gets removed (or when `git merge` runs from inside it), the shell dies permanently — every subsequent command returns exit code 1 with no recovery.

**Root cause from session `dbd6592f`:** The session ran `cd .worktrees/skill-improvements/dashboard && go build` to compile Go code. This moved the shell's cwd into the worktree. Later, `git merge && git worktree remove` ran from that same shell — `git merge` failed because cwd was inside the worktree, then every subsequent command failed too.

**Repeated in session `91d04155`:** Ironically, while rewriting the worktree tool in Go to prevent this exact bug. The session used `cd .worktrees/worktree-go-rewrite/.../scripts && go build ./...`. When the worktree became orphaned (concurrent session interference), it ran `rm -rf` on the worktree directory — but the shell CWD was still inside it from the earlier `cd`. Every subsequent command failed.

**Why it keeps happening:** The Bash tool maintains a **persistent shell** across calls. `cd` in one Bash tool call drifts the CWD for ALL future calls in that session. This is invisible — there's no prompt showing the CWD, and it's easy to forget a `cd` from 20 tool calls ago. The danger isn't the `cd` itself, it's the temporal distance between the `cd` and the later removal/cleanup that kills the shell.

**Prevention:** Never `cd` into a worktree. Always use:
1. `go run -C <worktree-skill>/scripts . exec <name> <cmd>` — uses `cmd.Dir`, parent CWD unchanged
2. Subshells: `(cd /path && cmd)` — CWD resets when subshell exits
3. Tool flags: `git -C <path>`, `go -C <path>`
4. Absolute paths: `go build /absolute/path/to/pkg/...`

## `git -C` does NOT protect the shell

`git -C <path>` changes git's working directory, not the shell's. If the shell cwd is inside a worktree, `git -C /project-root worktree remove` will still kill the shell because the shell's cwd gets deleted.

## Rebase from outside a worktree = fatal error

`git rebase main <branch>` checks out `<branch>` in the current tree, which fails with `fatal: '<branch>' is already used by worktree`. Use `git -C <worktree-path> rebase main` instead.

## Concurrent manager merges cause double-work

When multiple managers merge to main simultaneously, the second merge's rebase can silently overwrite the first manager's edits in shared files. In session `219ef347`, both liveness and session-id managers had their worktree edits reverted during concurrent merges and had to re-apply all changes — effectively doubling their work and cost.

**Mitigation:** The worktree CLI now acquires a lockfile (`.worktrees/.merge-lock`) before merging, making concurrent merges structurally impossible. If a merge is already in progress, subsequent attempts fail with a clear error. Stale locks (from crashed processes) are detected via PID and cleaned up automatically.

## Parallel test cleanup can race in ad-hoc git repos

`worktree_test` mostly uses `setupTestRepo()` because it includes robust cleanup (`git worktree prune`, removing `.git/worktrees`, removing `.worktrees`).

An ad-hoc test that created a repo via `t.TempDir()+git init` showed intermittent cleanup failure under package-wide parallel load:
- `TempDir RemoveAll cleanup: unlinkat .../.git: directory not empty`

Using `setupTestRepo()` for that case removed the flake while keeping behavior coverage.

See also [[delegation/include-domain-quirks]], [[principles/fix-root-causes]], [[principles/encode-lessons-in-structure]]
