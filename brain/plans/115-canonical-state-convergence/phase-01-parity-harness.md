Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 1 — Parity Harness

## Goal

Create the minimum proof layer the migration needs: canonical execution and restart behavior must be measurable against the current loop behavior before any caller cutover starts, without building a large standalone test program that delays deletion work.

## Changes

- **`internal/integration/resilience_test.go`** — add legacy-vs-canonical parity coverage for representative order lifecycles and crash windows
- **`internal/projection/projection_test.go`** — compare projection output where projection parity is the right signal
- **`loop/testdata/` + `loop/fixture_test.go`** — add only the permanent fixtures needed for restart/adoption/merge-review invariants

## Data structures

- `parityCase` — fixture/test record describing lifecycle inputs, crash point, and expected durable state markers
- `parityResult` — normalized end-state summary covering projection output plus recovery-relevant canonical metadata

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Test strategy and migration-boundary judgment |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/integration/... ./internal/projection/... ./loop/...`
- `go vet ./internal/integration/... ./internal/projection/... ./loop/...`

### Runtime
- prove that a representative execute/review/merge flow reaches equivalent final projected output and equivalent recovery-relevant canonical metadata through both paths
- prove that interrupted writes and replay windows still converge to one end state
- record only the permanent fixtures for adopted sessions, pending review, merge recovery, and scheduler special cases so later phases cannot regress them silently
- treat `pnpm test:smoke` as a phase gate even this early; unexpected smoke failures block the phase
