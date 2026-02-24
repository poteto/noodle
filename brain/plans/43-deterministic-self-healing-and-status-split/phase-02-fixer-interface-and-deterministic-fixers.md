Back to [[plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 2: Fixer interface and deterministic fixers

## Goal

Create a registry of deterministic repair functions that fix known-shape issues instantly without spawning an agent. Most fixers touch only `.noodle/` state files. Two fixers (orphaned worktrees, stale locks) mutate `.worktrees/` — these are higher-risk and must be conservative (only act on provably orphaned state). No fixer touches source code or creates new worktrees. Each fixer is a small, testable, idempotent function.

## Changes

### New package (`loop/repair/`)

- `Fixer` — function type: `func(ctx FixContext) (fixed bool, err error)`
- `FixContext` — carries paths (runtime dir, project dir, queue file, etc.) and a logger interface for recording what was fixed (for future TUI feed integration, todo #44)
- `RunAll(ctx FixContext, fixers []Fixer)` — runs fixers in order, returns which ones fired

### Fixer implementations (each a separate file in `loop/repair/`)

1. **`fix_malformed_queue.go`** — If queue.json exists but fails JSON parse, reset to `{"items":[]}`. Prioritize will regenerate next cycle. Idempotent: valid JSON is a no-op.

2. **`fix_malformed_mise.go`** — If mise.json exists but fails JSON parse, delete it. The mise builder regenerates it every cycle. Idempotent: missing or valid file is a no-op.

3. **`fix_missing_dirs.go`** — Ensure `.noodle/`, `.noodle/sessions/` exist via `os.MkdirAll`. Idempotent: already-existing dirs are a no-op.

4. **`fix_stale_sessions.go`** — Scan `.noodle/sessions/*/meta.json`. For any session with status `running`/`spawning`, determine liveness by runtime type:
   - **tmux sessions**: Check if the tmux session exists via `tmux has-session -t <name>`. **Important**: tmux session names use the `noodle-` prefix transform from `tmuxSessionName()` (`reconcile.go:143`, `dispatcher/tmux_command.go:110`), not the raw session ID. The fixer must apply the same transform: `"noodle-" + sanitizeSessionToken(sessionID)`. This is the same mechanism `loop/reconcile.go:40-49` already uses.
   - **sprites sessions**: Check the `Alive` field in meta.json (set by the monitor). If `Alive` is false and status is still `running`, mark as `exited`.
   - **Unknown runtime / no runtime field**: Skip — don't mark as exited if we can't determine liveness. Leave for agent judgment.

   Note: session meta has no PID field, so PID-based liveness is not available. Each runtime has its own liveness signal. Idempotent: already-exited sessions are a no-op.

5. **`fix_orphaned_worktrees.go`** — Run `git worktree list`, find worktrees under `.worktrees/`. A worktree is orphaned only if **all** of: (a) no session's `spawn.json` in `.noodle/sessions/` references its path (worktree linkage lives in `spawn.json` at `worktree_path`, not in `meta.json` — see `cook.go:315-322`), (b) no session's `meta.json` has a non-terminal status (`running`/`spawning`/`stuck`). This is a `.worktrees/` mutation, not a `.noodle/` mutation — it carries more risk than state file fixes. Run `git worktree remove` only on true orphans. Idempotent: no orphans means no-op.

6. **`fix_stale_locks.go`** — Check `.worktrees/.merge-lock`. If the PID in the lock file is dead, remove it. Reuse the `isProcessAlive()` pattern from `worktree/lock.go`. This is a `.worktrees/` mutation. Idempotent: no lock or live PID is a no-op.

### PID liveness helper

`isProcessAlive()` currently lives in `worktree/lock.go`. Either:
- Move to a shared `internal/procutil/` package (preferred — both worktree and repair need it)
- Or duplicate the 10-line function (acceptable if import would create a cycle)

## Data structures

- `Fixer` function type
- `FixContext` struct with paths and logger
- `FixResult` — which fixers ran and what they fixed (for logging/future TUI feed)

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Each fixer is a small self-contained function with clear spec |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Unit test each fixer in isolation:
  - Write malformed JSON to a temp queue file, run fixer, assert valid empty queue
  - Write malformed JSON to temp mise file, run fixer, assert file deleted
  - Create a meta.json with status "running" but no real process, run fixer, assert status "exited"
  - Create a merge lock with a dead PID, run fixer, assert lock removed
- Test idempotency: run each fixer twice in sequence, assert same result
