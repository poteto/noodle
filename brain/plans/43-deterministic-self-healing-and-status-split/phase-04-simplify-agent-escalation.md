Back to [[plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 4: Simplify agent escalation

## Goal

Redesign the agent repair path in `runtime_repair.go`. Simplify state tracking (replace fingerprinting with plain keys, simplify adoption to a lightweight scan). Keep what works (attempt limits, loop pausing, dispatcher integration).

## Changes

### Simplify fingerprinting in `runtime_repair.go`

- `runtimeIssueFingerprint()` — SHA1 hashing of scope+message+warnings. Replace with `scope + "|" + message` as the dedup/attempt key. Using scope alone would risk false fatal escalation when unrelated errors hit the same scope (e.g. two different `loop.spawn` failures). Including the message distinguishes them without the SHA1 overhead.

### Simplify adoption (keep, don't delete)

The current `findRunningRuntimeRepairSessionID()` is 40 lines of directory scanning and JSON parsing. This can't be fully removed — if the loop restarts while a repair agent is still running (tmux session alive), the stale-session fixer correctly skips it (it's alive), so without adoption the loop would spawn a *duplicate* repair agent for the same issue.

**Keep a lightweight reattach**: on startup, scan `.noodle/sessions/` for any session whose ID starts with `repair-runtime-` and whose meta.json status is `running`/`spawning`. If found, set `runtimeRepairInFlight` with that session ID and pause the loop. This is the same logic as today but without the `Fingerprint` field and with simpler state tracking.

**Delete `adoptRunningRuntimeRepair()`** as a separate method — fold the scan into `ensureRuntimeRepair` as a first check before spawning.

### Simplify `runtimeRepairState` in `loop/types.go`

Remove `Fingerprint` field. Keep `SessionID` for the adoption/reattach case (loop restarted, no `Session` object available). The struct becomes:

- `Key string` — dedup key (`scope|message`)
- `Issue runtimeIssue` — for retry prompt
- `Attempt int`
- `SessionID string` — for reattached sessions (no `Session` object)
- `Session dispatcher.Session` — for freshly spawned sessions
- `StateBefore State`

### Simplify `ensureRuntimeRepair`

- Attempt tracking uses `scope|message` as key instead of fingerprint hash
- Reattach check: if `runtimeRepairInFlight` is nil, scan for running repair sessions before spawning new one
- Rest stays the same: max 3 attempts, pause loop, spawn via dispatcher

### Simplify `advanceRuntimeRepair`

- Keep both the `Session.Done()` path (freshly spawned) and the `SessionID`-only path (reattached) — the latter reads meta.json status to determine completion
- Simplify the SessionID path: just check meta.json status, don't rebuild state from scratch

### Rename `handleRuntimeIssue` → `escalateToAgent`

Clearer name now that it's only called after deterministic fixes fail. Called by `triageAndRepair` from phase 3.

## Data structures

- `runtimeRepairState` — simplified (see above)
- `runtimeRepairAttempts map[string]int` — keyed by `scope|message` string instead of fingerprint hash

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Judgment calls on what to keep vs delete, ensuring no behavioral regressions |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Trigger an issue that fixers can't resolve — verify agent spawns, loop pauses, loop resumes after completion
- Trigger the same issue 4 times — verify max attempts limit (3) still works, 4th attempt returns fatal error
- Restart with live repair agent — loop restarts, finds a still-running `repair-runtime-*` session, reattaches instead of spawning a duplicate
- Restart with dead repair agent — loop restarts, finds a stale `repair-runtime-*` session (tmux dead), stale-session fixer marks it exited, fresh repair spawns if issue persists
