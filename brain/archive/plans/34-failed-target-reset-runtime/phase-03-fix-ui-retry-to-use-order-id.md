Back to [[plans/34-failed-target-reset-runtime/overview]]

# Phase 3: Fix UI Retry to Use Order ID

## Goal

Ensure failed-item retry sends a true order ID so `requeue` succeeds reliably.

## Changes

- `dispatcher/types.go` + `dispatcher/dispatch_metadata.go`
  - Thread `order_id` into spawn metadata at dispatch time.
- `loop/cook.go` and `loop/schedule.go` dispatch request construction
  - Populate dispatch request order ID for all stage/session spawns.
- `internal/snapshot/snapshot.go` + `internal/snapshot/types.go`
  - Read `order_id` from spawn metadata and expose on snapshot `Session`.
- `ui/src/client/types.ts` + `ui/src/components/DoneCard.tsx`
  - Use `session.order_id` for failed retry action.
  - If missing, avoid sending an invalid `requeue` payload.
- Tests in `dispatcher/*_test.go` and `internal/snapshot/snapshot_test.go` for `order_id` round-trip.

## Data Structures

- Add explicit `order_id` field to dispatch/spawn metadata and snapshot session DTO.
- Maintain single source of truth: order identity should come from spawn metadata, not parsed naming conventions.

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Why: cross-language but still deterministic plumbing work.

## Verification

Static:
- `go test ./dispatcher ./internal/snapshot`
- `pnpm test:short`

Runtime:
- Force a failed stage; click Retry in UI.
- Confirm `/api/control` receives `order_id` that exists in `failedTargets` and requeue succeeds.
