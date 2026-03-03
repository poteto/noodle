# Phase 03: WS Lifecycle Fix

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

No panics, no leaks, one frontend connection. Fix the bounded set of crash/leak risks in the WS transport layer without refactoring the broader UI.

## Findings in scope

- `69`, `70`, `76`, `77`, `83`

## Priority

- **P0 first:** `69` (send-on-closed-channel panic), `83` (ReviewList crash on stale index).
- **P1 next:** `70` (subscriber leak), `76` (duplicated frontend WS lifecycle), `77` (control fallback stall).

## Changes

- Make server WS hub safe for concurrent close/broadcast (fix send-on-closed-channel).
- Add explicit unsubscribe cleanup on client disconnect (fix subscriber leak).
- Replace duplicated per-hook WS connection managers with single connection owner.
- Fix control command fallback to not stall on WS timeout before HTTP fallback.
- Guard ReviewList against stale selected index after list mutation.

## Done when

- No send-on-closed-channel panic path remains.
- Subscriber count matches active client count after disconnect cycles.
- Frontend uses one WS connection per app instance.
- ReviewList selection is bounds-checked after list changes.

## Verification

### Static
- `go test ./server/...`
- `go test -race ./server/...`
- `pnpm --filter noodle-ui typecheck`
- `pnpm --filter noodle-ui test`

### Runtime
- Disconnect/reconnect cycles with active subscribers — assert zero panics, zero leaked subscribers.
- Control command under WS-unavailable state — assert HTTP fallback within bounded time.

## Rollback

- Server WS and frontend connection changes are independently revertible.
