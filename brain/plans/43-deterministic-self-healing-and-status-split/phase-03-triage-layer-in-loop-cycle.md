Back to [[plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 3: Triage layer in loop cycle

## Goal

Wire the fixer registry into the loop's error handling path. When an issue is detected, try deterministic fixes first. Only escalate to agent repair if fixers don't resolve it.

## Changes

### New triage function (`loop/repair.go` or inline in `loop/loop.go`)

- `triageAndRepair(ctx, scope, err, warnings)` — replaces `handleRuntimeIssue` as the entry point:
  1. Run applicable fixers via `repair.RunAll()` — no new worktree created, no source code touched (fixers operate on `.noodle/` state files and `.worktrees/` cleanup only)
  2. If any fixer reports `fixed`, return nil — the loop's existing cycle will naturally re-run the failing operation on the next iteration (the loop already retries every cycle tick)
  3. If no fixer matched or none resolved, fall through to agent escalation (the simplified version from phase 4) — only agent escalation creates a worktree

The triage function does not re-run the failing operation itself. It returns nil to let the loop's normal cycle retry. This keeps the signature simple — no retry callback needed.

### Update all error paths to route through triage

Seven `handleRuntimeIssue` call sites (lines 209, 212, 215, 230, 247, 257, 315) switch to `triageAndRepair`. The function signature stays the same: `(ctx, scope, err, warnings)`.

**Additionally**, the `readQueue` parse error at line 242-244 currently returns immediately (`return Queue{}, false, err`) without ever hitting the repair path. This is the primary malformed-queue failure path. Change this to route through triage:

```go
queue, err := readQueue(l.deps.QueueFile)
if err != nil {
    return Queue{}, false, l.triageAndRepair(ctx, "loop.queue.parse", err, nil)
}
```

This is an 8th call site, not a replacement of an existing `handleRuntimeIssue` call.

### Fixer selection by scope

Not all fixers apply to all scopes. The triage function runs fixers relevant to the scope:

| Scope | Applicable fixers |
|-------|-------------------|
| `loop.queue.parse` | `fixMalformedQueue` |
| `loop.queue` | (none — validation errors need agent judgment) |
| `mise.build` | `fixMalformedMise`, `fixMissingDirs` |
| `mise.sync` | `fixMissingDirs` |
| `loop.collect` | `fixStaleSessions`, `fixMalformedQueue` |
| `loop.monitor` | `fixStaleSessions` |
| `loop.spawn` | `fixOrphanedWorktrees`, `fixStaleLocks` |
| `loop.control` | `fixMissingDirs`, `fixMalformedQueue` |

**Note on `fixMalformedQueue` scope**: `readQueue` is called from multiple paths beyond the primary `prepareQueueForCycle` parse — including `controlEnqueue` (`control.go:250`), `skipQueueItem` (`cook.go:406`), and `lookupQueueItem` (`cook.go:302`). A malformed queue will surface as errors in `loop.control` or `loop.collect` scopes before ever reaching `loop.queue.parse`. Therefore `fixMalformedQueue` must be available to those scopes too.

Any scope can also run `fixMissingDirs` as a baseline.

## Data structures

No new types — uses `repair.Fixer` and `repair.FixContext` from phase 2.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Wiring existing functions into existing call sites |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Corrupt queue.json, trigger a cycle — verify Go fixes it without spawning an agent and without creating a worktree
- Delete `.noodle/sessions/` dir, trigger a cycle — verify it's recreated, no worktree created
- Corrupt queue.json in a way that a fixer can't help (e.g. valid JSON but invalid task keys) — verify agent escalation fires and only then is a worktree created
