---
todo: 105
---

# Phase 2 — Remove Ticket Staleness Tracking

Back to [[plans/102-config-cleanup/overview]]

## Goal

Remove the concept of ticket staleness. The config field `monitor.ticket_stale` is defined but never wired to production code. The staleness logic in `event/tickets.go` uses a hardcoded constant. Both should go — the scheduler sees stage events and decides what to do about stuck work.

**Critical prerequisite:** Before removing staleness, verify that orphaned ticket claims (cook crashes after `ticket_claim`, before `ticket_done/release`) are handled by session reconciliation or another crash-recovery path. If staleness is the *only* mechanism that makes crash-orphaned orders re-dispatchable, replace it with a simpler claim-expiry reaper tied to session liveness (session dead → release its claims) before deleting `TicketStatusStale`.

## Changes

### `config/types_defaults.go`
- Remove `TicketStale string` from `MonitorConfig`

### `config/parse.go`
- Remove `monitor.ticket_stale` default-setting logic
- Remove `ticket_stale` duration validation

### `config/config_test.go`
- Remove tests asserting ticket_stale defaults and parsing

### `event/tickets.go`
- Remove `defaultTicketStaleTimeout` constant
- Remove staleness check logic from `Materialize()` (the part comparing `now - lastProgress` against stale timeout)
- Remove `TicketStatusStale` constant if it exists
- Keep the rest of the ticket materializer intact

### `docs/reference/configuration.md`
- Remove `ticket_stale` from monitor section docs

### `generate/skill_noodle.go`
- Remove `monitor.ticket_stale` field description

## Data Structures

Remove `TicketStatusStale` from any ticket status enum. The `Ticket` type may lose a staleness-related field if one exists.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Targeted deletion with clear spec |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `TicketStale`, `ticket_stale`, or `TicketStatusStale` remain (grep check)

### Runtime
- `go test ./event/...` — ticket materializer tests pass without staleness assertions
- `go test ./config/...` — config tests pass
- **Crash recovery test:** simulate cook crash after `ticket_claim` without `ticket_done`. Verify the order becomes re-dispatchable (either via session reconciliation or the replacement claim-expiry mechanism)
