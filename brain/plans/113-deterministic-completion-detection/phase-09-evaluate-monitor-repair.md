Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 9: Evaluate and Simplify Monitor Repair

## Goal

With the CompletionTracker providing reliable in-process completion detection and sprites heartbeats fixed, determine whether `session_meta_repair.go` (zombie repair) is still needed. If redundant, remove it. If still useful, simplify it.

## Changes

This is an **evaluation phase** — the implementation depends on findings.

**Evaluate:** With the new system, under what conditions can a session be "terminal by claims but alive by liveness"?
- processSession: `waitForExit()` blocks on process exit, then calls `markDone()`. The tracker resolves on exit. A zombie would mean the process wrote EventComplete but didn't exit — this is a provider bug, not a Noodle bug.
- spritesSession: `waitAndSync()` blocks on `sprites.Cmd.Wait()`, then calls `markDone()`. Same logic.

**If zombies are still possible** (provider process hangs after emitting completion):
- Keep `enqueueTerminalActiveCompletions()` but simplify: use `session.Outcome().Status.IsTerminal()` instead of re-reading meta.json and re-deriving status from claims
- The monitor repair becomes a simple "is the session's outcome terminal but the process is still alive? Kill it."

**If zombies are no longer possible** (CompletionTracker + process exit guarantee no zombie state):
- Delete `loop/session_meta_repair.go`
- Remove `enqueueTerminalActiveCompletions()` call from the monitor pass pipeline
- Simplify `monitor/derive.go` to remove zombie-detection logic

**Likely outcome:** Keep as simplified defense-in-depth. Provider processes can hang (OOM, deadlock, infinite loop after emitting completion), and the repair layer catches this cheaply.

## Data Structures

- No new types

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (evaluation requires judgment)

## Verification

- If kept: verify simplified repair still catches zombie scenario (test with mock session that emits completion but doesn't exit)
- If removed: verify no test regressions, ensure all zombie scenarios are handled by other mechanisms
- `go test ./loop/... ./monitor/... -race`
