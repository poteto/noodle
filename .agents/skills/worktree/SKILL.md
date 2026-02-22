---
name: worktree
description: >
  Manage git worktrees for isolated branch work without disrupting the current working tree.
  Use when the user says "worktree", "new worktree", "create a worktree", "clean up worktrees",
  or wants to work on a separate branch in isolation. Also use when spawning agent teams that
  need file-level isolation to work in parallel without conflicts. Also triggers on session
  startup when CLAUDE.md requires worktree isolation for concurrent sessions.
---

# Worktree

**Default to linked-worktree isolation.** Multiple sessions often run concurrently. Setup costs ~30 seconds and prevents hours of merge pain.

## Valid Workspace Definition

- **Valid for significant or autonomous work**: linked checkout created via Skill(worktree) `create` (typically `.worktrees/<name>`)
- **Not valid by default**: primary checkout at repo root
- **Allowed exception**: interactive, single-agent, small changes can run in primary checkout when that is the deliberate choice

All operations use `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree` — it enforces CWD safety, correct sequencing, stash/rebase handling, and dep reinstall.

## Commands

### Preflight

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree list
```

Shows all worktrees with merge status. Run at session start to spot stale worktrees from crashed sessions.

### Create

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree create <name>
```

Creates worktree + branch at `.worktrees/<name>`, symlinks `.claude/settings.local.json`, auto-detects package manager, installs deps.

### Run commands inside a worktree

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree exec <name> <command...>
```

Runs the command inside the worktree via `cmd.Dir`. **The parent shell's CWD never changes** — this is the safe way to run builds, tests, or any tool that needs the worktree as CWD.

```bash
# Examples
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree exec my-feature go test ./...
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree exec my-feature go build ./...
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree exec my-feature go test ./...
```

For git operations, `git -C` also works and doesn't need `exec`:

```bash
git -C <project-root>/.worktrees/<name> status
git -C <project-root>/.worktrees/<name> add -A
git -C <project-root>/.worktrees/<name> commit -m "message"
```

File reads/edits use absolute paths directly — no `exec` needed.

### Merge back to main

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree merge <name>
```

Single command does: CWD safety check -> stash worktree noise -> rebase onto integration branch -> merge -> remove worktree -> delete branch -> prune -> reinstall deps. Refuses to run if CWD is inside the worktree or if root checkout is not on the integration branch.

By default, integration branch is auto-detected from `origin/HEAD` (falls back to `main` if detection fails).

If there are no commits to merge, it exits early and leaves the worktree/branch in place so you can decide whether to continue work or run cleanup.

Merges are serialized via lockfile (`.worktrees/.merge-lock`). If another merge is in progress, the command waits with short retry intervals (default timeout: 5 minutes) and then proceeds automatically. Stale locks from crashed processes are detected and cleaned up automatically.

### Prune merged/patch-equivalent worktrees

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree prune
```

Auto-cleans safe worktrees:
- no commits ahead of integration branch, or patch-equivalent commits only
- clean working tree (no uncommitted changes)
- removes stale `.worktrees/<name>` directories that are no longer real worktrees

### Clean up without merging

```bash
go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree cleanup <name>
```

Warns if there are unmerged commits. Use `--force` to discard them.

## Critical Safety Rules

**NEVER `cd` into `.worktrees/`** — if the shell CWD is inside a worktree when it gets removed, the shell dies permanently (exit code 1, no recovery, session over). This applies to:

- Direct `cd /path/to/worktree && ...` in Bash tool calls
- Any command that changes CWD as a side effect

The Bash tool maintains a persistent shell — `cd` in one call persists to ALL future calls. A later `rm -rf` or `git worktree remove` on that path kills the shell instantly.

**Always use one of these instead:**
1. `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree exec <name> <cmd>` — CWD-safe by design
2. Subshells: `(cd /path && cmd)` — CWD resets when subshell exits
3. Absolute paths: `go build /path/to/pkg/...`
4. Tool flags: `git -C <path>`, `go -C <path>`

## Parallel Agent Teams

1. Lead creates one worktree per teammate: `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree create phase-1`
2. Spawn each with `mode: "acceptEdits"` — worktrees are isolated scratch space
3. Each teammate commits on its own branch
4. Lead merges each: `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree merge phase-1`

**Foundational changes**: If Phase 1 is a foundation later phases need, do it on main first, commit, then create parallel worktrees. Branches start from the commit they were created at.

**When not to use**: sequential tasks, read-only research, single-agent work with no concurrency risk.

## Conventions

- **Directory**: `.worktrees/` at project root (gitignored)
- **Naming**: `.worktrees/<branch-name>`
- **Branches are ephemeral** — merge back to main and delete when done
- **Brain vault is shared state** — never branch it. Teammates send results back to the lead.

## Stale Worktree Triage

Use `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree list` to check status, then:

1. **Merged** (shows "safe to clean up") -> `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree cleanup <name>`
2. **Unmerged with commits** -> report to user, let them decide merge vs discard
3. **No commits** -> crashed session artifact -> `go run -C $CLAUDE_PROJECT_DIR/old_noodle . worktree cleanup <name>`
