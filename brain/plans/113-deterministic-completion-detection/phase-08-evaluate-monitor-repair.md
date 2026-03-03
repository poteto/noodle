Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 8: Evaluate and Simplify Monitor Repair

## Goal

With the CompletionTracker providing reliable in-process completion detection and sprites heartbeats fixed, determine whether `session_meta_repair.go` (zombie repair) is still needed. If redundant, remove it. If still useful, simplify it.

## Changes

**Two concrete changes**, then simplification evaluation.

### Fix 1: Claims must detect Claude terminal state

**`monitor/claims.go`**: Currently `claims.Completed` is only set on `EventComplete` (`claims.go:89`). Claude never emits `EventComplete` — only `EventResult`. A Claude session that emits `EventResult` then hangs forever will never trigger zombie repair. Fix: set `claims.Completed = true` when `EventResult` is seen (a completed turn is a terminal signal for zombie detection purposes).

### Fix 2: ForceKill race

Repair calls `ForceKill()` which sets status to "killed" and wakes the watcher, creating a race between the repair's "completed" completion and the watcher's "cancelled" completion. Fix: repair should enqueue its completion BEFORE calling ForceKill. The generation counter in `applyStageResult` deduplicates, so the watcher's later completion is harmlessly dropped.

### Simplification evaluation

**Zombie detection must remain claims-based.** `session.Outcome()` is only populated after `Done()` closes, but the zombie case is "process alive, terminal events emitted." `Outcome()` returns zero value for zombies because `Done()` hasn't closed.

Keep `enqueueTerminalActiveCompletions()` with claims-based detection. Simplify the *response* logic but keep detection external.

**Likely outcome:** Keep as simplified defense-in-depth with both fixes above. Provider processes can hang (OOM, deadlock, infinite loop after emitting completion).

## Data Structures

- No new types

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (evaluation requires judgment)

## Verification

- If kept: verify simplified repair still catches zombie scenario (test with mock session that emits completion but doesn't exit)
- If removed: verify no test regressions, ensure all zombie scenarios are handled by other mechanisms
- `go test ./loop/... ./monitor/... -race`
