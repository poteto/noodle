Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 9: Evaluate and Simplify Monitor Repair

## Goal

With the CompletionTracker providing reliable in-process completion detection and sprites heartbeats fixed, determine whether `session_meta_repair.go` (zombie repair) is still needed. If redundant, remove it. If still useful, simplify it.

## Changes

This is an **evaluation phase** — the implementation depends on findings.

**Evaluate:** With the new system, under what conditions can a session be "terminal by claims but alive by liveness"?
- processSession: `waitForExit()` blocks on process exit, then calls `markDone()`. The tracker resolves on exit. A zombie would mean the process wrote EventComplete but didn't exit — this is a provider bug, not a Noodle bug.
- spritesSession: `waitAndSync()` blocks on `sprites.Cmd.Wait()`, then calls `markDone()`. Same logic.

**Zombie detection must remain claims-based.** `session.Outcome()` is only populated after `Done()` closes, but the zombie case is "process alive, terminal events emitted." `Outcome()` returns zero value for zombies because `Done()` hasn't closed. Claims-based detection (reading canonical events from disk, independent of session state) is the correct mechanism.

**Simplification targets:**
- Keep `enqueueTerminalActiveCompletions()` with claims-based detection
- Simplify the *response*: use tracker-derived status mapping where possible, but detection stays external
- Fix the ForceKill race: repair calls `ForceKill()` which sets status to "killed" and wakes the watcher, creating a race between the repair's "completed" completion and the watcher's "cancelled" completion. Fix: repair should enqueue its completion BEFORE calling ForceKill, or ForceKill on a session with an already-queued completion should be a status no-op.

**Likely outcome:** Keep as simplified defense-in-depth with the ForceKill race fixed. Provider processes can hang (OOM, deadlock, infinite loop after emitting completion).

## Data Structures

- No new types

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (evaluation requires judgment)

## Verification

- If kept: verify simplified repair still catches zombie scenario (test with mock session that emits completion but doesn't exit)
- If removed: verify no test regressions, ensure all zombie scenarios are handled by other mechanisms
- `go test ./loop/... ./monitor/... -race`
