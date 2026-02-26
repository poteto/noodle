Back to [[plans/34-failed-target-reset-runtime/overview]]

# Phase 1: Make Failed Target Reload Correct and Explicit

## Goal

Make runtime failed-target state reflect disk truth during loop execution, including deletions from `.noodle/failed.json`.

## Changes

- `loop/failures.go`
  - Split parsing from mutation: read/parse failed-target map from disk into a fresh map.
  - Replace in-memory `l.failedTargets` snapshot on reload (do not merge into existing map).
- `loop/loop.go`
  - Include `failed.json` in fsnotify trigger handling.
  - Reload failed targets when `failed.json` changes before spawn planning.
- `loop/loop_test.go` and/or `loop/control_test.go`
  - Add regression coverage for runtime `failed.json` edits and deletion behavior.

## Data Structures

- Keep `failedTargets map[string]string` as the authoritative in-memory set.
- Introduce a reload helper that returns a full snapshot map to replace current state atomically.

## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Why: clear implementation against existing loop/state architecture.

## Verification

Static:
- `go test ./loop`
- `go vet ./loop`

Runtime:
- Start loop with one failed order in `.noodle/failed.json`.
- While loop runs, remove that key from file.
- Confirm next cycle dispatches the previously blocked order without restart.
