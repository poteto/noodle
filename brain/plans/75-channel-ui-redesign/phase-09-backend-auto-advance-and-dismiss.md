Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 8: Backend — Stage Messages and Auto-advance

## Goal

When a stage completes, it may optionally emit a message for the scheduler. If no message: auto-advance (the loop handles it, no scheduler involvement). If a message is present: forward it to the scheduler, which decides what to do — advance, retry, add stages, park for review.

This replaces the hardcoded quality verdict mechanism. Quality verdict files, the Go code that reads and interprets them, and the approve-mode parking logic are all removed. The quality skill becomes just another agent that sends a message to the scheduler with its findings. The scheduler — a persistent live agent with context about the whole order — makes the call.

## Prerequisites

**Persistent scheduler session.** This phase depends on the scheduler being a persistent live agent (established in phase 2). The current `steer("schedule", ...)` path in `cook_steer.go:41` calls `rescheduleForChefPrompt`, which rewrites `orders.json` to a single schedule order — it does NOT send a message to a live session. Phase 2 must establish: (1) the scheduler as a persistent session that stays alive across cycles, and (2) a direct message path from the loop to the scheduler via `controller.SendMessage()`, bypassing the `rescheduleForChefPrompt` shortcut.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Architectural change to loop lifecycle, needs careful reasoning about concurrency and event flow

## What already exists (do NOT reimplement)

- **`noodle event emit`**: `cmd_event.go` — CLI command that emits loop events to `loop-events.ndjson`. Extend this to support session events so agents can emit `stage_message` from the command line. No file-based messaging needed — the CLI handles atomic NDJSON writing.
- **Event reader**: `event/reader.go` — `EventReader.ReadSession()` reads session events with filters. The loop already uses this for steer resume context (`cook_steer.go:144`).
- **Loop event emitter**: `event/loop_event.go` — `LoopEventWriter.Emit()` appends lifecycle events. The `stage.completed` event already has `StageCompletedPayload`.
- `schedule` is already special-cased to auto-advance on success (`cook_completion.go:124-129`) — this special case gets removed since auto-advance is now the default for all messageless completions.

## Key design decisions

1. **Auto-advance is the default.** No message = advance. The loop handles this with no scheduler involvement. This is fast and has no dependency on the scheduler being alive.
2. **Messages use the session event system via CLI.** Agents call `noodle event emit --session $SESSION_ID stage_message --payload '...'` to emit a session event. The loop reads it on completion via `EventReader`. Extend `cmd_event.go` to support a `--session` flag that writes to the session event log instead of the loop event log.
3. **Messages are forwarded to the scheduler via direct message.** The loop sends the message to the persistent scheduler session via `controller.SendMessage()` — NOT via `rescheduleForChefPrompt`. This requires the steer path for schedule targets to be updated in phase 2.
4. **The scheduler chat UI shows the message.** The `stage.completed` loop event carries the message payload. The feed renders it in the scheduler channel so the user can see what agents reported and what the scheduler decided.

## Subtract

- `loop/cook_completion.go` — remove `readQualityVerdict()` (lines 197-213) and the quality verdict gate in the auto-mode path (lines 140-155)
- `loop/control.go` — remove the quality verdict gate in `controlMerge` (lines 275-299)
- `loop/types.go` — remove `QualityVerdict` struct (lines 83-87)
- `loop/event_payloads.go` — remove `QualityWrittenPayload` (lines 66-71)
- `event/loop_event.go` — remove `LoopEventQualityWritten` constant. Also clean up references in schedule skill docs (`schedule/SKILL.md:76`) and mise event summaries (`mise/builder.go:313`).
- `loop/cook_completion.go` — remove the schedule special case (lines 124-129). Schedule is just another messageless stage that auto-advances. **Prerequisite**: `permissions.merge: false` must be set in schedule frontmatter first, otherwise schedule completion falls into the merge path with an empty worktree name.
- `loop/cook_completion.go` — remove approve-mode parking of ALL non-schedule stages (line 135). Parking is now the scheduler's decision when it receives a message, not a hardcoded loop behavior.
- `.noodle/quality/` directory and verdict files — no longer written or read

## Add

- `cmd_event.go` — add `--session <id>` flag to `noodle event emit`. When provided, writes to the session event log (`sessions/{id}/events.ndjson`) via `EventWriter` instead of the loop event log.
- `event/types.go` — add `EventStageMessage EventType = "stage_message"` to session event types
- `loop/cook_completion.go` — in `handleCompletion`, after stage completes: read session events for `stage_message`. If none found, auto-advance. If found and `blocking: true`, forward to scheduler and don't advance. If found and `blocking: false`, auto-advance AND forward message to scheduler (informational).
- `loop/event_payloads.go` — add `Message *string` field to `StageCompletedPayload` so the feed event carries the agent's message

## Modify

- `.agents/skills/quality/SKILL.md` — instead of writing a verdict file, call `noodle event emit --session $SESSION_ID stage_message --payload '...'` with the assessment content.
- `.agents/skills/schedule/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
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
- Don't add a `"review"` stage status — the scheduler parks reviews via a new `park-review` control command (see phase 9)
- Don't touch ticket protocol events — they're orthogonal

## Stage message schema

The `stage_message` event payload. Create a reference doc at `.agents/skills/quality/references/stage-message-schema.md` (quality-local, since the global references dir doesn't exist).

```json
{
  "message": "string — content for the scheduler, natural language",
  "blocking": "boolean — if true, scheduler must decide before stage advances. Omit or null = true"
}
```

- **`blocking: true` (or omitted)** — loop forwards message to scheduler, does NOT auto-advance. Scheduler must issue a control command (advance, add stage, etc.) before the pipeline continues.
- **`blocking: false`** — loop auto-advances AND forwards the message to the scheduler for information. The scheduler sees it but doesn't need to act.

Examples:
- Quality accept: `{ "message": "All checks pass. Tests green, scope clean.", "blocking": false }`
- Quality reject: `{ "message": "Rejected: 3 high issues. [1] Missing test for edge case in handleCompletion. [2] Scope violation: modified cook_merge.go outside plan phase scope. [3] Error message uses expectation form.", "blocking": true }`
- Execute complete: `{ "message": "Implementation complete. 3 files changed, 2 new tests added. Ready for review.", "blocking": true }`

Go type in `event/types.go`:
```go
type StageMessagePayload struct {
    Message  string `json:"message"`
    Blocking *bool  `json:"blocking,omitempty"`
}

func (p StageMessagePayload) IsBlocking() bool {
    if p.Blocking == nil {
        return true // default: blocking
    }
    return *p.Blocking
}
```

### Quality skill update

Update `.agents/skills/quality/SKILL.md`:
- Remove: "Write verdict to `.noodle/quality/<session-id>.json`"
- Add: "Emit a `stage_message` event: `noodle event emit --session $SESSION_ID stage_message --payload '<json>'`"
- Accept → emit `{ "message": "<summary>", "blocking": false }` — pipeline continues, scheduler sees the green light
- Reject → emit `{ "message": "<detailed findings>", "blocking": true }` — scheduler reads findings and decides (retry, add oops stage, or abort)
- The message content replaces the verdict JSON. Write it as natural language the scheduler can act on, not structured JSON. Include specific file paths, line context, and principle violations so the scheduler can brief the retry cook.

Update `.agents/skills/quality/references/verdict-schema.md` → rename to `stage-message-schema.md`, replace content with the generic schema above.

## Data Structures

- `EventStageMessage EventType = "stage_message"` — new session event type
- `StageMessagePayload` — the event payload (message + `*bool` blocking)
- `StageCompletedPayload.Message` — optional message field on the loop event payload
- Remove `QualityVerdict`, `QualityWrittenPayload`

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: `noodle event emit --session <id> stage_message` writes to session event log
- Unit test: stage with no `stage_message` event auto-advances (no scheduler involvement)
- Unit test: stage with blocking `stage_message` forwards to scheduler, does NOT auto-advance
- Unit test: stage with `blocking: false` auto-advances AND forwards message
- Unit test: omitted `blocking` field defaults to true
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
