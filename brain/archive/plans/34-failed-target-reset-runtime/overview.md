---
id: 34
created: 2026-02-26
status: ready
---

# Failed Target Reset Runtime

## Context

Todo #34 calls out a real runtime gap: failed targets are loaded at startup and cached in memory, so editing `.noodle/failed.json` during a running loop does not take effect until restart. We also found a related correctness issue: the web retry action currently sends session ID as `order_id`, but failed-target keys are order IDs.

## Scope

In scope:
- Make failed-target state reloadable during runtime (without restart).
- Add an explicit operator command to clear failed targets (`clear-failed`).
- Fix web retry wiring to send a real order ID, not a session ID.
- Add focused tests across loop, server, dispatcher/snapshot, and UI typing boundaries.

Out of scope:
- Redesigning broader failure lifecycle semantics (OnFailure pipeline, retry strategy).
- New UI surfaces beyond correcting retry behavior.
- Changing scheduler semantics beyond failed-target unblocking.

## Constraints

- Cross-platform behavior (macOS/Linux/Windows); no shell-specific runtime assumptions.
- No backward-compat dual paths by default.
- Keep error messages state-descriptive.
- Preserve existing failed-target stickiness semantics unless explicitly cleared/requeued.

Design-space alternatives considered:
1. Watch `failed.json` only.
2. Add `clear-failed` command only.
3. Do both watcher reload + explicit clear command.

Chosen: **(3)**. Watcher reload handles external/manual edits immediately; `clear-failed` provides an explicit, scriptable control surface.

Retry identity alternatives:
1. Parse order ID from session/worktree naming conventions.
2. Persist explicit `order_id` in spawn metadata and expose it in snapshot sessions.

Chosen: **(2)** for correctness and future-proofing.

## Applicable Skills

- `go-best-practices`
- `testing`
- `ts-best-practices`

## Phases

1. [[plans/34-failed-target-reset-runtime/phase-01-make-failed-target-reload-correct-and-explicit]]
2. [[plans/34-failed-target-reset-runtime/phase-02-add-clear-failed-control-command]]
3. [[plans/34-failed-target-reset-runtime/phase-03-fix-ui-retry-to-use-order-id]]
4. [[plans/34-failed-target-reset-runtime/phase-04-verify-with-focused-loop-server-and-ui-tests]]

## Verification

- `go test ./loop ./server ./dispatcher ./internal/snapshot`
- `go test ./...`
- `pnpm test:short`
- Runtime check: start loop, mark an order failed, edit/clear `.noodle/failed.json` while loop is running, verify dispatch unblocks without restart.
