Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 3: Consolidate State Utilities

## Goal

Eliminate three verbatim-duplicated functions across `internal/dispatch` and `internal/reducer` by moving them to `internal/state` where they belong. Per migrate-callers-then-delete-legacy-apis: add to state, migrate callers, delete originals.

## Changes

### `internal/state/state.go` — add four utilities:

1. **`(State).Clone() State`** — deep-copy method. Merge the two existing `cloneState` implementations. The `reducer.go` version is the superset (handles `ModeTransitions`), so use that as the base.

2. **`(OrderLifecycleStatus).IsTerminal() bool`** — method on the status type. Replaces `isTerminalOrder` duplicated in dispatch, reducer, and state.

3. **`(StageLifecycleStatus).IsTerminal() bool`** — method on the status type. Replaces `isTerminalStage` / `isTerminalStageStatus`.

4. **`(StageLifecycleStatus).IsBusy() bool`** — method on the status type. Replaces `isBusyStageStatus`. **Critical: busy and terminal are distinct concepts.** `IsBusy()` covers `dispatching|running|merging|review`. `pending` is non-terminal but NOT busy. Do not conflate them — dispatch capacity depends on this distinction (`internal/dispatch/dispatch.go:49`).

5. **`(State).LookupStage(orderID string, stageIndex int) (OrderNode, StageNode, bool)`** — method on State. Replaces `lookupOrderStage` duplicated in dispatch and reducer.

### Migrate callers:

- **`internal/dispatch/dispatch.go`** — replace `cloneState()`, `isTerminalOrder()`, `isTerminalStage()`, `lookupOrderStage()` calls with `state.State` methods. Delete the local functions. **Replace `isBusyStageStatus` calls with `IsBusy()`, NOT `!IsTerminal()`.**
- **`internal/reducer/reducer.go`** — same migration. Delete local functions.
- **`internal/state/state.go`** — replace `isTerminalStageStatus` / `isBusyStageStatus` with the new methods.
- **`internal/integration/resilience_test.go`** — has test copies of terminal checks; update to use the shared methods.

## Data Structures

- `State.Clone() State` — method, not standalone function
- `OrderLifecycleStatus.IsTerminal() bool` — method on existing type
- `StageLifecycleStatus.IsTerminal() bool` — method on existing type
- `StageLifecycleStatus.IsBusy() bool` — method on existing type (distinct from IsTerminal)
- `State.LookupStage(string, int) (OrderNode, StageNode, bool)` — method on existing type

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Judgment needed: merging two `cloneState` implementations requires understanding which fields each version handles.

## Verification

### Static
- `go build ./internal/...` — all references resolve
- `go test ./internal/state/...` — new methods have tests
- `go test ./internal/dispatch/...` — existing tests pass with new call sites
- `go test ./internal/reducer/...` — same
- `go vet ./...` — clean

### Runtime
- `go test ./...` — full suite passes
- Grep for `func cloneState`, `func isTerminalOrder`, `func isTerminalStage`, `func lookupOrderStage` — zero results outside `internal/state`
- **Clone immutability tests** — mutate a clone, assert the original is unchanged for: metadata maps, attempt exit code pointers, mode transition slices, and nested stage slices. These must exist before migrating callers.
- **Busy/terminal regression test** — prove that `pending` stages are not classified as busy (protects dispatch capacity logic)
