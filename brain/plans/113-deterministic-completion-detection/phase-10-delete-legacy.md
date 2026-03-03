Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 10: Delete Legacy Code

## Goal

Remove all legacy completion detection code that the CompletionTracker replaces. Clean migration — no backward compatibility shims.

## Changes

**`dispatcher/process_session.go`**:
- Delete `terminalStatus()` method (replaced by CompletionTracker.Resolve in phase 4)
- Remove `readCanonicalEvents()` call from exit path (already done in phase 4, verify no remnants)

**`dispatcher/session_helpers.go`**:
- Evaluate `readCanonicalEvents()` — if only used by the now-deleted `terminalStatus()`, delete it. If still used by other callers (monitor, stage_yield), keep it.

**`dispatcher/types.go`**:
- Remove `Status() string` from `Session` interface (replaced by `Outcome()`)
- Migrate all callers of `session.Status()` to use `session.Outcome().Status.String()`

**`loop/cook_watcher.go`**:
- Update `stageResultStatus()` to accept `SessionOutcome` instead of raw string
- Simplify: `outcome.Status` → `StageResultStatus` mapping is direct, no normalization needed

**`dispatcher/process_session_test.go`**:
- Delete `TestProcessSessionTerminalStatus*` tests (logic now tested via CompletionTracker tests from phase 2)

## Data Structures

- No new types — deletion only

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- `go build ./...` — no compilation errors after removing Status() from interface
- `go test ./... -race` — full test suite passes
- `pnpm check` — full suite
- Verify no remaining references to `terminalStatus`, `readCanonicalEvents` (if deleted), or `Status() string` on Session interface
