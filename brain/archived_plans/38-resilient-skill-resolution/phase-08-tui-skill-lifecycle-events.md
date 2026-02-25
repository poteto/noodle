Back to [[archived_plans/38-resilient-skill-resolution/overview]]

# Phase 8: TUI feed events for skill lifecycle

**Routing:** `codex` / `gpt-5.3-codex` — mechanical TUI integration, clear patterns to follow

## Goal

Surface skill lifecycle events (dropped queue items, registry rebuilds, bootstrap status) in the TUI feed so the user knows what's happening. Follow the existing steer event pattern: write events to NDJSON file, read during snapshot, render as feed cards.

## Changes

**Read queue events in TUI snapshot:**
- `tui/model_snapshot.go` — new `readQueueEvents()` function (modeled on `readSteerEvents()` at line 627). Reads `.noodle/queue-events.ndjson`, converts each line to a `FeedEvent`.
- `tui/model_snapshot.go` — call `readQueueEvents()` from `buildFeedEvents()` to merge queue events into the feed timeline.

**Feed event rendering:**
- `tui/feed_item.go` — handle new event categories:
  - `"queue_drop"` — amber/warning border, shows which item was dropped and why
  - `"registry_rebuild"` — neutral border, shows skills added/removed
  - `"bootstrap"` — brand color border, shows "creating prioritize skill from workflow analysis"
- `tui/styles.go` — add color/label mappings for new categories in `eventLabel()`

**Event producers (defined in earlier phases, consumed here):**
- Phase 5: `auditQueue()` writes `queue_drop` and `registry_rebuild` events
- Phase 7: bootstrap session completion handler writes `bootstrap` event
- All events share `queue-events.ndjson` and the `QueueAuditEvent` struct

**Dispatch warnings — feed events, not status line:**
- When a skill resolution soft-fail occurs (phase 2), the warning is already included in session events via `loadedSkill.Warnings`. These surface in the feed naturally as session event cards — no separate status line wiring needed. Drop the status line requirement; feed events are the consistent notification path.

## Quality reference inventory for this phase

These `quality` references are user-facing TUI/schema strings and should be
reviewed while wiring skill-lifecycle feed visibility:

- TUI labels/descriptions: `tui/config_tab.go`, `tui/verdict.go`
- TUI theme and task-type rendering: `tui/styles.go`, `tui/components/theme.go`
- TUI task-type fixtures: `tui/components/components_test.go`, `tui/queue_test.go`
- Shared schema-facing wording: `internal/schemadoc/specs.go`

Phase acceptance for this inventory:
- User-facing text should reflect the post-merge semantics (review flow) while
  still preserving `.noodle/quality/` verdict data contracts where required.

## Data structures

- `QueueAuditEvent` (from phase 5) — read by TUI
- New `FeedEvent.Category` values: `"queue_drop"`, `"registry_rebuild"`, `"bootstrap"`

## Verification

```bash
go test ./tui/... && go vet ./...
```

Unit tests:
- Write sample `queue_drop`, `registry_rebuild`, and `bootstrap` events to `queue-events.ndjson`. Build snapshot. Verify each appears as a FeedEvent with correct category.
- Verify feed card rendering: correct border colors, labels, body text for each category.
- Verify `readQueueEvents` handles malformed lines gracefully (skip, don't crash).

Manual: trigger a skill deletion while loop is running. Verify queue drop notification appears in TUI feed with amber border.
