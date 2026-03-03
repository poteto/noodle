# Phase 04: Server and UI WebSocket Convergence

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Converge backend WS lifecycle guarantees and frontend WS consumption into one coherent, race-safe transport model.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-03-loop-runtime-control-safety]]

## Findings in scope

- `69-70`, `76-88`

## Forward-design constraint

Phase 03 modifies server shutdown and session lifecycle behavior. This phase's WS client lifecycle state machine must build on Phase 03's shutdown semantics. Sketch the WS state machine design during Phase 03 execution to avoid redesigning what Phase 03 just changed.

## Within-phase priority

- **P0 first:** `69` (WS send-on-closed-channel panic), `83` (ReviewList crash on stale index).
- **P1 next:** `70`, `76`, `77`, `84`, `85`.
- **P2 last:** `78`, `79`, `80`, `81`, `82`, `86`, `87`, `88`.

## Changes

- Make server client lifecycle safe for concurrent close/broadcast and explicit unsubscribe cleanup.
- Introduce single frontend WS connection manager with explicit ownership/refcount semantics.
- Consolidate route/channel/bootstrap mapping and reduce duplicated UI state logic.
- Add protections for known crash-prone UI paths (review selection, schedule routing, key stability).

## Data structures

- WS client lifecycle state machine (server + frontend).
- Typed channel/route mapping and session-status union.
- Shared feed/render contract structures.

## Done when

- No send-on-closed-channel panic path remains in WS hub.
- Frontend uses a single connection lifecycle per app instance.
- Route/channel semantics are defined in one typed mapping path.
- Known crash-path regressions have test coverage.

## Verification

### Static
- `pnpm --filter noodle-ui typecheck`
- `pnpm --filter noodle-ui test`
- `go test ./server`

### Runtime
- Disconnect/reconnect chaos checks with active subscribers and snapshot broadcasts.
- Control command fallback checks under WS unavailable/opening states.

## Rollback

- Server WS and frontend WS manager changes remain independently revertible.
- Revert UI mapping changes separately if navigation regressions appear.
