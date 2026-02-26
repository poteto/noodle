Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 9: Integration tests at scale

## Goal

Prove the redesign works at scale with integration tests that simulate 100+ concurrent sessions. Verify cycle time, memory usage, completion throughput, crash recovery, and shutdown quiescence at the target scale.

## Changes

**`loop/integration_test.go`** — Add scale tests using a `MockRuntime` that simulates session dispatch and completion with configurable latency. Test scenarios:

1. **Throughput test**: Dispatch 100 orders with 3 stages each, complete them with random latency (100ms-2s), verify all orders complete and merge. Assert cycle time stays under 200ms.
2. **Burst completion test**: 50 sessions complete in the same cycle. Verify the completions channel drains correctly and all stages advance. No blocked senders.
3. **Crash recovery test**: Dispatch 50 sessions, kill the loop, restart, verify all active stages are re-derived from orders.json and orphaned sessions are recovered via `Runtime.Recover()`. Stages in `"merging"` state are handled correctly.
4. **Mixed runtime test**: Dispatch sessions across tmux and mock-cloud runtimes simultaneously, verify routing and completion work for both.
5. **Merge queue throughput**: 20 concurrent merge requests, verify serialization and no data loss. Context cancellation stops the queue within 5s.
6. **Shutdown quiescence test**: Start 20 sessions, initiate drain, verify all completions and merge results are processed before exit. No goroutine leaks.
7. **Goroutine leak test**: Start and stop the loop 3 times in sequence. Verify goroutine count returns to baseline after each stop. Covers: completion watchers, health observers, merge queue worker, runtime observation goroutines.
8. **Paused loop + full channel test**: Pause the loop, complete 100 sessions (filling channels), resume. Verify no data loss and no blocked goroutines.

**`runtime/mock_runtime.go`** (or test helper) — A mock Runtime implementing the full interface (`Start`, `Dispatch`, `Kill`, `Recover`, `Health`, `Close`) with configurable behavior per dispatch. Supports simulating failures, stuck sessions, health events, and pre-existing sessions for recovery testing.

**`loop/bench_test.go`** (new) — Benchmark the cycle at various session counts (10, 100, 500, 1000) to establish baselines and verify O(events + orders) scaling.

## Data structures

- `MockRuntime` — implements full `Runtime` interface with configurable behavior per dispatch
- `MockSessionHandle` — controllable `Done()` channel, configurable status/cost

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — test design requires understanding the full system to write meaningful assertions

## Verification

### Static
- `go test ./...` — all tests pass including new scale tests
- `go test -bench ./loop/...` — benchmarks produce cycle time numbers

### Runtime
- Throughput test passes: 100 orders × 3 stages complete in <60s test time
- Burst test passes: 50 completions in one cycle processed correctly
- Crash test passes: kill-9 and restart converges to correct state
- Shutdown test passes: all goroutines exit cleanly, no leaked watchers
- Goroutine leak test passes: count returns to baseline after each stop
- Benchmark shows cycle time <200ms at 1000 sessions
- Race detector clean: `go test -race ./loop/... ./runtime/...`
