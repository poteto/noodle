Back to [[plans/88-subagent-tracking-v2/overview]]

# Phase 3: Ingestion Lifecycle Reliability

## Goal

Make ingestion lifecycle-safe across start, stop, restart, and crash-recovery paths.

## Changes

- Add bounded polling scheduler for out-of-band sources with explicit interval and max sources per session.
- Wire ingestion workers to session lifecycle cancellation so all goroutines and file handles close on stop.
- Add restart reconciliation that reattaches known active sources and cleans stale checkpoints.

Poller operating envelope:
- Default poll interval: 1s (configurable)
- Max sources per parent session: 64
- Per-tick processing budget: 500 lines total (carry remainder to next tick) to avoid starvation/backpressure collapse

## Data Structures

- `IngestionHandle`: `{SessionID, Cancel, Done, Sources}`
- `IngestionRegistry`: map keyed by parent session ID for active workers + checkpoints + poller metrics

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Lifecycle scaffolding and concurrency correctness.

## Verification

### Static
- `go test ./dispatcher/... ./loop/...`
- `go test -race ./dispatcher/... ./loop/...`

### Runtime
- Start/stop sessions repeatedly; verify no leaked workers and no stale emits after stop.
- Crash/restart simulation; verify reattach behavior and checkpoint continuity.
