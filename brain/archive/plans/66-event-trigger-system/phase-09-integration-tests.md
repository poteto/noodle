Back to [[archive/plans/66-event-trigger-system/overview]]

# Phase 9 — Integration Tests

## Goal

End-to-end verification that events flow from emission through to the mise brief. Covers the full pipeline: state transition → EventWriter → NDJSON file → mise builder → brief. Also covers external event injection.

## Changes

- **`loop/loop_event_integration_test.go`** (new) or add to existing loop test files — integration test that:
  1. Sets up a Loop with EventWriter
  2. Drives a cook through completion (stage advance)
  3. Reads `loop-events.ndjson` and verifies `stage.completed` event with correct payload
  4. Builds a mise brief and verifies `recent_events` contains the event
  5. Drives a schedule completion, rebuilds brief, verifies watermark advanced (previous events excluded)

- **`loop/loop_event_integration_test.go`** — additional cases:
  - Failure path: cook fails → `stage.failed` + `order.failed` events appear
  - Merge path: merge completes → `worktree.merged` event appears
  - Control command: requeue → `order.requeued` event appears
  - Watermark: after `schedule.completed`, only newer events in brief
  - External event: `EventWriter.Emit` with arbitrary type (e.g., `ci.failed`) appears in brief alongside internal events

## Data Structures

None — tests only.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./loop/... -run Integration && go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

- All integration tests pass
- Full test suite still passes (no regressions)
- Lint passes
- No references to `queue-events.ndjson` or `QueueAuditEvent` remain in non-archived source code (grep verification — exclude `brain/archive/plans/` and `brain/plans/` directories)
