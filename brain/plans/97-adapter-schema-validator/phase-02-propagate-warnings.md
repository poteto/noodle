Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 2: Propagate Warnings Through Runner and Mise Builder

## Goal

Wire the new warnings return from `ParseBacklogItems` through `Runner.SyncBacklog()` and into `Builder.Build()` so adapter validation warnings reach `mise.Brief.Warnings`.

## Changes

### `adapter/runner.go`

- `SyncBacklog` signature: `([]BacklogItem, []string, error)` — passes through parse warnings
- Call `ParseBacklogItems`, collect both items and warnings, return both

### `mise/builder.go`

- In `Build()`, where `b.runner.SyncBacklog()` is called (line 48): capture warnings from the new return value
- Append adapter warnings to the existing `warnings` slice (which already handles "sync script missing")
- No change to `Brief` struct — `Warnings []string` field already exists

### Tests

- `adapter/runner_test.go` — update `SyncBacklog` test to check warnings propagation
- `mise/builder_test.go` (or inline test) — verify that adapter warnings flow into `brief.Warnings`

## Data structures

No new types. `SyncBacklog` return changes to match `ParseBacklogItems`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical plumbing, follows existing patterns |

## Verification

### Static
- `go test ./adapter/... ./mise/...` passes
- `go vet ./adapter/... ./mise/...` clean

### Runtime
- Test: adapter with one bad item → `SyncBacklog` returns valid items + warnings
- Test: builder receives adapter warnings → `brief.Warnings` includes them alongside any builder-level warnings
- Test: adapter with no issues → empty warnings, same behavior as before
