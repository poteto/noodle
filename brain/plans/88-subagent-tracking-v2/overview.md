---
id: 88
created: 2026-03-01
status: active
---

# Plan 88: Sub-Agent Tracking V2

Back to [[plans/index]]

## Context

Plan 84 establishes sub-agent visibility and steering. V2 focuses on deferred capabilities and reliability hardening: out-of-band ingestion, stronger identity reconciliation, lifecycle cleanup guarantees, and robust control-path steering behavior under failure and restart conditions.

## Scope

**In scope:**
- Out-of-band ingestion for Codex child session files and Claude team inbox files
- Canonical agent identity hardening across all spawn/progress/complete paths
- Session lifecycle-safe ingestion (startup, shutdown, restart, orphan cleanup)
- Steering hardening through existing `/api/control` command path (`steer-agent`)
- Snapshot/UI reconciliation for out-of-order and late events
- Hardening fixtures and integration tests for full in-band + out-of-band behavior

**Out of scope:**
- New event pipeline or alternate websocket protocol
- New per-agent HTTP endpoint outside `/api/control`
- Provider SDK integrations (Claude/Codex SDK)
- Cross-session orchestration UX beyond a single Noodle session tree/feed

## Constraints

- Preserve existing canonical pipeline: `parse.CanonicalEvent -> event.Event -> WebSocket -> UI`
- Canonical `AgentID` must always be stable provider identity (no provisional ID persistence)
- Out-of-band ingestion must be bounded and lifecycle-safe (no unbounded watcher loops)
- Control behavior must remain centralized in loop command processing
- Keep provider-specific parsing at boundaries; internal snapshot/UI logic remains provider-agnostic

## Alternatives Considered

1. **Unbounded fsnotify watchers for provider directories** — low initial latency, but high lifecycle/race risk and cleanup complexity. Not chosen.
2. **Bounded poller + checkpointed offsets (chosen)** — deterministic lifecycle, restart-safe, easier verification, acceptable latency tradeoff.
3. **Rebuild ingest around provider SDK callbacks** — potentially cleaner long-term, but out of current scope and increases integration surface.

## Applicable Skills

- `go-best-practices` — backend ingestion, lifecycle, and control-path phases
- `testing` — fixture and integration hardening phases
- `react-best-practices` — UI reconciliation and optimistic state handling
- `ts-best-practices` — typed event/state narrowing for agent-specific UI paths

## Phases

1. [[plans/88-subagent-tracking-v2/phase-01-out-of-band-agent-ingestion]]
2. [[plans/88-subagent-tracking-v2/phase-02-agent-identity-reconciliation-hardening]]
3. [[plans/88-subagent-tracking-v2/phase-03-ingestion-lifecycle-reliability]]
4. [[plans/88-subagent-tracking-v2/phase-04-steer-agent-control-hardening]]
5. [[plans/88-subagent-tracking-v2/phase-05-snapshot-ui-consistency]]
6. [[plans/88-subagent-tracking-v2/phase-06-hardening-test-matrix]]

## Verification

```bash
go test ./... && go vet ./...
go test -race ./...
sh scripts/lint-arch.sh
cd ui && pnpm tsc --noEmit && pnpm test
```
