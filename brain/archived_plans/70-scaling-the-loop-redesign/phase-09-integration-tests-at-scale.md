Back to [[archived_plans/70-scaling-the-loop-redesign/overview]]

# Phase 9: Integration tests at scale

## Goal

Prove the redesign works at scale with integration tests that simulate 100+ concurrent sessions. Verify cycle time, memory usage, completion throughput, crash recovery, and shutdown quiescence at the target scale.

## Changes

**`loop/integration_test.go`** — Add scale tests using a `MockRuntime` (implementing the `Runtime` interface from phase 3) that simulates session dispatch and completion with configurable latency. Note: the `runtime/` package and `Runtime` interface are created in phase 3 — this phase depends on P3 being complete. All test scenarios use the mock runtime's `Dispatch`/`Kill`/`Recover`/`Health`/`Close` methods, not the `dispatcher.Dispatcher` interface. Test scenarios:

1. **Throughput test**: Dispatch 100 orders with 3 stages each, complete them with fixed-seed pseudo-random latency (100ms-2s), verify all orders complete and merge. Record p95/p99 cycle time and compare to baseline (no hard CI threshold).
2. **Burst completion test**: 200 sessions complete in the same cycle with tiny injected completion buffer/overflow limits. Verify exact-once drain and all stages advance. No blocked senders.
3. **Crash recovery test**: Dispatch 50 sessions, kill the loop, restart, verify all active stages are re-derived from orders.json and orphaned sessions are recovered via `Runtime.Recover()`. Stages in `"merging"` state are handled via deterministic recovery algorithm (phase 7): check merge metadata in `Extra`, verify branch ancestry, re-enqueue or advance accordingly.
4. **Mixed runtime test**: Dispatch sessions across tmux and mock-cloud runtimes simultaneously, verify routing and completion work for both.
5. **Merge queue throughput**: 20 concurrent merge requests, verify serialization and no data loss. Context cancellation stops the queue within 5s.
6. **Shutdown quiescence test**: Start 20 sessions, initiate drain, verify `watcherWG` completes, merge queue reaches `Pending()==0` and `InFlight()==0`, all completions and merge results are processed before exit. No goroutine leaks.
7. **Goroutine leak test**: Start and stop the loop 3 times in sequence. Verify goroutine count returns to baseline after each stop. Covers: completion watchers, health observers, merge queue worker, runtime observation goroutines.
8. **Paused loop + full channel test**: Pause the loop, complete 100 sessions (filling overflow slices), resume. Verify no data loss and no blocked goroutines.
9. **Manual + automatic merge interleaving test**: Dispatch 5 sessions that complete, simultaneously issue `controlMerge` for 3 pending-review orders. Verify all 8 merges serialize correctly through the queue, no deadlock, no data loss.
10. **Server snapshot under load test**: With 100 active sessions, hit `/api/snapshot` endpoint 50 times concurrently. Verify all responses are valid JSON, contain all required fields, and respond in <100ms. Hit `/api/sessions/{id}/events` for 10 different sessions — verify lazy file reads return correct events.
11. **Dispatch fallback test**: Primary runtime `Dispatch` fails; verify one-time fallback dispatch via tmux runtime succeeds.
12. **Runtime contract race test**: Concurrent natural completion + `Kill()` + `Close()` still closes `Done()` exactly once.
13. **Stale merge result test**: generation N merge result arrives after generation N+1 for same `(order, stage)`; N is ignored.
14. **Control replay crash-window test**: crash between `orders.json` and `last-applied-sequence` write; restart replays without duplicate enqueue.

**`runtime/mock_runtime.go`** (or `loop/mock_runtime_test.go` test helper) — A mock Runtime implementing the full `Runtime` interface (`Start`, `Dispatch`, `Kill`, `Recover`, `Health`, `Close`) with configurable behavior per dispatch. Supports simulating failures, stuck sessions, health events, and pre-existing sessions for recovery testing. Place in `loop/` test files if the `runtime` package doesn't export test helpers.

**`loop/bench_test.go`** (new) — Benchmark the cycle at various session counts (10, 100, 500, 1000) to establish baselines and verify O(events + orders) scaling. Use `go test -bench=. ./loop/...` (not `-bench ./loop/...` which is invalid syntax).

## Data structures

- `MockRuntime` — implements full `Runtime` interface with configurable behavior per dispatch
- `MockSessionHandle` — controllable `Done()` channel, configurable status/cost

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — test design requires understanding the full system to write meaningful assertions

## Verification

### Static
- `go test ./...` — all tests pass including new scale tests
- `go test -bench=. ./loop/...` — benchmarks produce cycle time numbers

### Runtime
- Throughput test: 100 orders × 3 stages complete in <60s wall time, all orders reach `"completed"` status (not `"done"` — current terminal status is `"completed"` per `loop/types.go:40-44`), merge queue processes all requests
- Burst test: 200 completions in one cycle with tiny buffers — verify exact-once drain semantics, all stages advance to next
- Crash test: kill loop with 50 active sessions → restart → `Runtime.Recover()` called, all active stages either re-attached or reset to pending. Stages in `"merging"` state recovered via deterministic algorithm (phase 7)
- Shutdown test: `goleak.VerifyNone(t)` (add `go.uber.org/goleak` dependency with ignore policy for known runtime goroutines) — zero leaked goroutines. All watcher goroutines, health observers, merge worker, and runtime observation goroutines exit
- Goroutine leak test: `runtime.NumGoroutine()` at baseline, after 3 start/stop cycles, delta is 0 (±2 for runtime internals)
- Benchmark: cycle time at 1000 sessions — record p99 latency and compare to baseline trend (don't assert hard threshold in CI). Memory growth over 10,000 cycles — record and compare, don't hard-assert.
- Race detector clean: `go test -race ./loop/...` with all scale tests enabled (add `./runtime/...` when package exists)
- Server integration: `/api/snapshot` responds correctly under concurrent load with in-memory state
- Server integration: `/api/sessions/{id}/events` returns correct per-session events via lazy read
- Fallback integration: runtime dispatch failure path routes to tmux fallback exactly once
- Runtime contract: `Done()` exact-once invariant holds under concurrent completion/kill/close
