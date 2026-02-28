Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 8: Backend — Stage Messages and Auto-advance

## Goal

When a stage completes, it may optionally emit a message for the scheduler. If no message: auto-advance (the loop handles it, no scheduler involvement). If a message is present: forward it to the scheduler, which decides what to do — advance, retry, add stages, park for review.

This replaces the hardcoded quality verdict mechanism. Quality verdict files, the Go code that reads and interprets them, and the approve-mode parking logic are all removed. The quality skill becomes just another agent that sends a message to the scheduler with its findings. The scheduler — a persistent live agent with context about the whole order — makes the call.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Architectural change to loop lifecycle, needs careful reasoning about concurrency and event flow

## What already exists (do NOT reimplement)

- **Event emitter**: `event/writer.go` — `EventWriter.Append()` writes NDJSON events per session. Agents already emit ticket protocol events (`ticket_claim`, `ticket_progress`, etc.) through this system. The stage message is a new event type in the same system.
- **Event reader**: `event/reader.go` — `EventReader.ReadSession()` reads session events with filters. The loop already uses this for steer resume context (`cook_steer.go:144`).
- **Steer mechanism**: `cook_steer.go` — `steer()` can send messages to live sessions via `controller.SendMessage()`. This is how the loop forwards messages to the scheduler.
- **Loop event emitter**: `event/loop_event.go` — `LoopEventWriter.Emit()` appends lifecycle events to `loop-events.ndjson`. The `stage.completed` event already has `StageCompletedPayload`.
- `schedule` is already special-cased to auto-advance on success (`cook_completion.go:124-129`) — this special case gets removed since auto-advance is now the default for all messageless completions.

## Key design decisions

1. **Auto-advance is the default.** No message = advance. The loop handles this with no scheduler involvement. This is fast and has no dependency on the scheduler being alive.
2. **Messages use the existing session event system.** Agents emit a `stage_message` event via `EventWriter`. The loop reads it on completion via `EventReader`. No new file format — just a new event type in the existing NDJSON log.
3. **Messages are forwarded to the scheduler via steer.** The loop calls `steer("schedule", message)` to inject the message into the scheduler's conversation. The scheduler LLM processes it and issues control commands.
4. **The scheduler chat UI shows the message.** The `stage.completed` loop event carries the message payload. The feed renders it in the scheduler channel so the user can see what agents reported and what the scheduler decided.

## Subtract

- `loop/cook_completion.go` — remove `readQualityVerdict()` (lines 197-213) and the quality verdict gate in the auto-mode path (lines 140-155)
- `loop/control.go` — remove the quality verdict gate in `controlMerge` (lines 275-299)
- `loop/types.go` — remove `QualityVerdict` struct (lines 83-87)
- `loop/event_payloads.go` — remove `QualityWrittenPayload` (lines 66-71)
- `event/loop_event.go` — remove `LoopEventQualityWritten` constant
- `loop/cook_completion.go` — remove the schedule special case (lines 124-129). Schedule is just another messageless stage that auto-advances.
- `loop/cook_completion.go` — remove approve-mode parking of ALL non-schedule stages (line 135). Parking is now the scheduler's decision when it receives a message, not a hardcoded loop behavior.
- `.noodle/quality/` directory and verdict files — no longer written or read

## Add

- `event/types.go` — add `EventStageMessage EventType = "stage_message"` to session event types
- `loop/cook_completion.go` — in `handleCompletion`, after stage completes: read session events for `stage_message`. If found, forward to scheduler via steer and don't advance. If not found, auto-advance.
- `loop/event_payloads.go` — add `Message *string` field to `StageCompletedPayload` so the feed event carries the agent's message

## Modify

- `.agents/skills/quality/SKILL.md` — instead of writing a verdict file to `.noodle/quality/`, emit a `stage_message` event with the verdict content. The message should include accept/reject and findings in natural language — the scheduler interprets it.
- `.agents/skills/schedule/SKILL.md` — add `permissions.merge: false` to noodle frontmatter (it doesn't produce a mergeable worktree)
- `.agents/skills/quality/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
- `.agents/skills/reflect/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
- `loop/cook_completion.go` — `handleCompletion` uses `StageResult.Status` instead of re-deriving from `cook.session.Status()`
- `internal/snapshot/snapshot.go` — `readLoopEvents` maps ALL defined loop event types (not just 5). Include message content from `StageCompletedPayload.Message` in the feed event body. Normalize scheduler feed identity to one canonical channel ID.

### Feed events — full mapping

Map all loop event types from `event/loop_event.go`:

- `stage.completed` — label: "Completed", body: agent message if present
- `stage.failed` — label: "Failed", body: reason
- `order.completed` — label: "Order Complete"
- `order.failed` — label: "Order Failed", body: reason
- `order.dropped` — already mapped
- `order.requeued` — label: "Requeued"
- `quality.written` — remove (quality is no longer special-cased)
- `schedule.completed` — label: "Scheduled"
- `worktree.merged` — label: "Merged"
- `merge.conflict` — label: "Conflict"
- `registry.rebuilt` — already mapped
- `bootstrap.completed` — already mapped
- `bootstrap.exhausted` — already mapped
- `sync.degraded` — already mapped

Also populate `FeedEvent.task_type` (currently always empty).

### What NOT to change

- Don't touch `advanceOrder` — it already handles the relevant stage statuses
- Don't touch the merge step itself — `canMerge` still determines whether a worktree gets merged
- Don't add a `"review"` stage status — the scheduler parks reviews via the existing `pending_reviews` mechanism
- Don't touch ticket protocol events — they're orthogonal

## Data Structures

- `EventStageMessage` — new session event type, payload: `{ "message": "string" }`
- `StageCompletedPayload.Message` — optional message field on the existing payload
- Remove `QualityVerdict`, `QualityWrittenPayload`

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: stage with no `stage_message` event auto-advances (no scheduler involvement)
- Unit test: stage with `stage_message` event forwards message to scheduler via steer, does NOT auto-advance
- Unit test: `readQualityVerdict` is gone — no quality verdict file reading
- Unit test: schedule stage auto-advances like any other messageless stage (no special case)
- Unit test: `readLoopEvents` maps all defined loop event types
- Unit test: `StageCompletedPayload.Message` is included in feed event

### Runtime
- Submit an order with schedule → execute → quality pipeline
- Schedule completes with no message → auto-advances
- Execute completes with no message → auto-advances (or with message if execute skill emits one)
- Quality completes with message ("accept" / "reject with findings") → forwarded to scheduler
- Scheduler reads quality message in its chat, decides next action
- Quality message visible in scheduler chat UI
