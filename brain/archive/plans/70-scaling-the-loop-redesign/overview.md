---
id: 70
created: 2026-02-26
status: done
---

# Scaling the Loop Redesign

## Context

Noodle currently handles 10-30 concurrent agents well. The loop polls all sessions every cycle, maintains per-session maps, reads per-session files, and rebuilds the brief from scratch each time. This works because tmux sessions are local and file I/O is fast.

Cloud dispatchers (Sprites, Cursor) already exist. The question is: what changes when a scheduler wants to spawn 1,000 cloud agents? Every O(n) per-session operation becomes a bottleneck. The loop spends its entire cycle scanning sessions instead of advancing orders.

The core insight: **the loop should be organized around orders, not sessions.** Sessions are an implementation detail of the dispatch layer. The loop cares about stage completion, not session liveness. Push-based completion (sessions signal when done) replaces pull-based scanning (loop checks every session every cycle).

This redesign makes the cycle O(events + orders), independent of total sessions.

## Scope

**In scope:**
- Eliminate per-session O(n) operations from the loop cycle
- Push-based completion via channels (replacing session polling)
- Pluggable Runtime interface abstracting tmux/cloud dispatch + observation
- In-memory order state with periodic flush (replacing per-event disk I/O)
- Aggregate mise brief (replacing per-session ActiveCooks enumeration)
- Async merge queue (decoupling merges from the cycle)
- Web UI reads from in-memory state via HTTP, not from session files

**Out of scope:**
- Self-replicating Noodle / multi-instance orchestration (future plan)
- New agent roles or coordination protocols
- Database or message queue dependencies
- Changes to the scheduling skill or order format
- Remote runtime implementation details (Sprites/Cursor API specifics)

## Constraints

- **Files as API**: Runtime state stays in JSON files. In-memory state is an optimization — files are the persistence layer, not a replacement target.
- **Zero new dependencies**: No databases, message queues, or distributed systems libraries.
- **Existing tests pass throughout**: Each phase must leave `go test ./...` green.
- **Crash safety preserved**: The loop must survive kill -9 and restart cleanly. Write-before-action semantics for dispatch, idempotent recovery on startup.
- **Local tmux still works**: Cloud scaling is additive. The default single-machine tmux experience is unchanged.

## Alternatives considered

1. **Self-replicating Noodle** — Parent spawns child Noodle instances, each managing 10-30 agents. Explored extensively but adds operational complexity (child lifecycle, shared merge lock, scoped briefs) that isn't needed when cloud runtimes already handle agent execution. Revisit if single-instance scaling hits a wall.

2. **Server-as-control-plane** — Agents send heartbeats to a central server. Rejected: agents don't heartbeat consistently, and that's something the deterministic layer must handle from the outside.

3. **Incremental patching** — Add caching/batching to existing pull-based loop. Rejected: the fundamental structure (loop organized around sessions) is the bottleneck. Per redesign-from-first-principles, redesign holistically.

## Applicable skills

- `go-best-practices` — Go concurrency patterns, lifecycle management
- `testing` — Fixture framework, integration tests
- `codex` — Mechanical refactoring phases (map removal, interface extraction)
- `execute` — Implementation methodology
- `frontend-design` — Stage rail UI component in phase 8

## Phases

- [[archive/plans/70-scaling-the-loop-redesign/phase-01-subtract-session-maps-and-decouple-order-state]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-02-push-based-completion-channels]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-03-pluggable-runtime-interface]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-04-in-memory-orders-with-periodic-flush]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-05-aggregate-mise-brief]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-06-platform-side-health-observation]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-07-async-merge-queue]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-08-snapshot-and-web-ui-decoupling]]
- [[archive/plans/70-scaling-the-loop-redesign/phase-09-integration-tests-at-scale]]

### Dependency DAG

Hard dependencies (must complete before):
```
P1 → P2 → P3 → P4 → P7
                ↓      ↓
               P6    P8 ← P5
                ↓
               P9 (depends on P2, P3, P4, P6, P7)
```

- **P5** and **P6** are independent after P3/P4 and can run in parallel
- **P7** should run before P5/P6 — it's higher risk and later phases benefit from merge queue existing
- **P8** depends on P4 (in-memory state) + P5 (ActiveSummary contract)
- **P9** depends on all loop-mechanics phases (P2-P7)

### Concurrency model at scale

The global `MaxCooks` config gates total concurrent dispatches. At cloud scale (100-1000 agents), this needs per-runtime concurrency limits: `config.Runtime.Sprites.MaxConcurrent`, `config.Runtime.Cursor.MaxConcurrent`, with `MaxCooks` as the global ceiling. Each runtime enforces its own cap in `Dispatch()`, and the loop enforces the global cap. Phase 3 should introduce per-runtime caps alongside the Runtime interface.

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Each phase has phase-specific verification. End-to-end: run the loop with a mock cloud runtime dispatching 100+ simulated sessions, record cycle-time p95/p99, and compare against a checked-in baseline in a dedicated perf job (no hard latency gate in default CI).
