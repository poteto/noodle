Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 7 — Legacy Deletion And Hardening

## Goal

Finish the convergence by deleting any remaining superseded lifecycle and snapshot paths, then lock in the canonical-only architecture with the permanent regression suite.

## Changes

- **`loop/`** — remove any remaining direct order mutation paths, planner helpers, or restart helpers that survived earlier phases
- **`server/` + `internal/snapshot/`** — remove compatibility shims left over from the snapshot cutover
- **tests/fixtures** — convert migration parity coverage into canonical-only regression coverage and delete migration-only scaffolding

## Data structures

- final authoritative boundaries: canonical state, projected files/full snapshots, runtime-only enrichment
- canonical-only regression fixture set replacing legacy-vs-canonical comparison harnesses

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Final deletion sequencing and architectural hardening |

## Verification

### Static
- `pnpm test:smoke`
- `pnpm check`
- `go test -race ./...`
- `go vet ./...`
- `sh scripts/lint-arch.sh`

### Runtime
- rerun the canonical-only regression suite and resilience tests with no legacy lifecycle or snapshot source available
- prove full schedule -> execute -> merge -> review/reject/recover flows through the canonical/projection path only
- verify restart, adoption, pending-review recovery, and merge recovery still converge with the legacy helpers removed
- rerun `pnpm test:smoke` on the legacy-free architecture and treat any failure as a release blocker
