Back to [[plans/34-failed-target-reset-runtime/overview]]

# Phase 2: Add Clear-Failed Control Command

## Goal

Add an explicit control command that clears all failed-target blocks in one action.

## Changes

- `loop/control.go`
  - Add `clear-failed` action branch in `applyControlCommand`.
  - Implement `controlClearFailed()` to clear `l.failedTargets` and persist empty `failed.json` atomically.
- `server/server.go`
  - Add `clear-failed` to `/api/control` action allowlist.
- `loop/control_test.go`
  - Add tests for successful clear behavior and persisted file update.
- `server/server_test.go`
  - Add control API acceptance test for `clear-failed`.

## Data Structures

- Reuse `ControlCommand.Action` string with new literal `clear-failed`.
- No new payload fields.

## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Why: mostly mechanical control-path extension with tests.

## Verification

Static:
- `go test ./loop ./server`
- `go vet ./loop ./server`

Runtime:
- Seed `.noodle/failed.json` with multiple entries.
- Send `{"action":"clear-failed"}` to `/api/control`.
- Confirm blocked orders become dispatchable on next cycle and file is emptied.
