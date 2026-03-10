Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 6: Hardening Test Matrix

## Goal

Add high-signal fixture and integration coverage for v2 behavior across providers, ingestion modes, lifecycle transitions, and steering outcomes.

## Changes

- Expand parse fixtures to include in-band + out-of-band mixed streams and identity edge cases.
- Add lifecycle tests for bounded pollers (start/stop/restart/orphan cleanup).
- Add steering tests for success/failure/retryable provider errors and UI optimistic rollback behavior.

## Data Structures

- Fixture matrix dimensions: `{Provider, SourceMode, EventOrder, LifecycleState, SteeringOutcome}`
- Expected assertions on canonical events, snapshot agent nodes, and UI event lines

Priority order:
1. P0 critical slices: identity correctness, steering delivery/typed failure, restart checkpoint correctness
2. P1 reliability slices: out-of-order events, bounded poller backpressure, lifecycle cleanup
3. P2 completeness slices: provider-specific edge formatting and rare mixed-mode combinations

Upgrade-path slices (required):
- P0: replay plan-84-era logs (in-band only, sparse agent metadata) under v2 code and assert no panics, stable tree materialization, and backward-compatible UI event filtering
- P1: mixed upgrade replay where legacy logs and new out-of-band events coexist for the same session

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Mechanical fixture/test authoring from clear scenarios.

## Verification

### Static
- `go test ./parse/... ./internal/snapshot/... ./loop/... ./server/...`
- `go test -race ./...`
- `cd ui && pnpm test`

### Runtime
- Run end-to-end fixture pipeline (parse -> event -> snapshot -> websocket -> UI filter) and verify assertions for each matrix slice.
- Verify cleanup assertions after forced stop/restart cycles.
- Execute P0 slices on every CI run; P1 on main branch CI; P2 nightly or pre-release.
