Back to [[plans/34-failed-target-reset-runtime/overview]]

# Phase 4: Verify with Focused Loop, Server, and UI Tests

## Goal

Prove the full end-to-end behavior and prevent regressions across boundaries.

## Changes

- Add/expand integration-style tests that cover:
  - Runtime reload of `failed.json` changes.
  - `clear-failed` control path from API to loop state.
  - UI retry payload correctness (`order_id`, not session ID).
- Tighten any fixtures or snapshot assertions needed for the new `order_id` session field.

## Data Structures

- Reuse phase-added `order_id` and `failedTargets` structures.
- No new core types.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Why: verification phase may require judgment on test breadth and edge-case completeness.

## Verification

Static:
- `go test ./loop ./server ./dispatcher ./internal/snapshot`
- `go test ./...`
- `pnpm test:short`

Runtime:
- Start app + web UI.
- Reproduce current failure: mark order failed, observe retry.
- Validate fixed behavior: retry succeeds, and manual file edits / `clear-failed` unblock without restart.
- Edge cases: empty `failed.json`, malformed `failed.json`, concurrent control command + file edit.
