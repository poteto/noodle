---
id: 66
created: 2026-02-27
status: done
---

# Event Trigger System

## Context

Skills can only run when the schedule skill explicitly queues them. There's no way to react to lifecycle events — a merge completing, a stage failing, a quality verdict landing. This forces all reactive behavior through a scheduling bottleneck that must anticipate every scenario upfront.

The loop already has natural event points (cook completion, merge, order advance, registry rebuild) but they're scattered across two disconnected systems: per-session `event.Event` records and a separate `QueueAuditEvent` NDJSON log. Neither supports reactive scheduling.

## Design

The vision says agents are the orchestrator. Skills are the extension point. The schedule agent already decides what to work on — it just needs better information.

**The design:** Go emits lifecycle events to an append-only NDJSON file (mechanics). Events flow into the mise brief (mechanics). The schedule agent sees what happened and decides how to react (judgment). No trigger matching in Go. No event→skill dispatch mechanism. No new frontmatter. The LLM decides, because that's what noodle does.

This means:
- Deploy-after-merge is a line in the schedule skill prompt, not a Go feature
- Notify-on-failure is the schedule agent seeing `stage.failed` events and creating a notification order
- Reactive scheduling is just scheduling with better context
- Every new reaction is a skill prompt change, not a code change

**No in-memory channel.** The NDJSON file is the single source of truth. The bus is just a writer with a sequence counter — no buffered channel, no in-memory drain. The mise builder reads the file when constructing the brief. This eliminates backpressure concerns and aligns with the "everything is a file" vision.

**Best-effort emission.** Event writes are best-effort: on write failure, log a warning and continue. This means events can be silently lost. The tradeoff is intentional — the loop must never crash or stall because of event infrastructure. The schedule agent already handles missing information gracefully (it schedules from incomplete state every cycle). Lossiness is acceptable because the agent converges on correct behavior over multiple cycles, not from any single event.

**Lossy by design.** The event file is truncated to the last 200 records on startup and the mise brief includes at most 50 post-watermark events. High-volume runs will drop older unreacted events. This is the correct tradeoff for a file-mediated system — if "react to every event" is ever required, it needs a cursor file with delivery guarantees, which is out of scope.

**Synergy with plan 48 (live agent steering):** Today, the schedule agent spawns per cycle. With plan 48's bidirectional pipes, the schedule agent stays alive between cycles and receives new prompts when mise.json updates. This eliminates event-reaction latency — the schedule agent is always warm. The event system works without plan 48 (next-cycle latency); plan 48 makes it instant.

## Scope

**In scope:**
- Loop event types and NDJSON writer with monotonic sequences
- Delete the `QueueAuditEvent` system and migrate `internal/snapshot` consumer
- Lifecycle event emission at every state-transition point (including control commands and crash recovery)
- Events surfaced in mise brief as `recent_events`, watermarked by sequence number
- NDJSON file truncation (cap at last N events)
- `noodle event emit` CLI command for external event injection
- Schedule skill updated to react to events (internal and external)

**Out of scope:**
- Cron/timer triggers (different emission source — follow-up)
- HTTP webhook receiver for push-based external events (follow-up — the CLI command covers pull/script-based ingestion)
- Trigger frontmatter on skills (the schedule agent handles dispatch, not Go code)
- UI event browser (can consume `loop-events.ndjson` when built)

**Subsumes:** #34 (failed target reset — `stage.failed` events visible in mise, `requeue` command exists; UI retry identity is not covered and remains as residual scope on #34 if needed)

**Related but not subsumed:** #29 (context passthrough part addressed — schedule agent has event context and sets Stage.Prompt; backlog-only scheduling simplification is a separate concern), #50 (reschedule button — reactive scheduling reduces the need, but a manual UI button is orthogonal if still desired)

## Constraints

- **Cycle-based model preserved.** Events are emitted at state transitions, surfaced in mise via file read, consumed by the schedule agent on its next run.
- **One event system.** `QueueAuditEvent` is deleted and replaced. No dual paths.
- **File is truth.** No in-memory event channel. NDJSON file is the single source. Watermark by monotonic sequence, not timestamp.
- **Idempotent.** Sequence-based watermark. Mise brief includes events since last schedule run's sequence. Crash-safe: sequence is persisted in the event record.
- **Agents decide.** Go code emits events and surfaces them. The schedule agent decides what to do. No Go code interprets event semantics.

## Alternatives Considered

1. **Mechanical triggers in Go.** Skills declare `triggers: ["stage.completed"]` in frontmatter; Go matcher dispatches. Rejected: Go code doing orchestration contradicts the vision. Skills are the extension point, agents are the orchestrator.
2. **In-memory event channel + drain.** Buffered channel with non-blocking send. Rejected: introduces backpressure/drop risk, the channel has no consumer in the schedule-agent design, and the NDJSON file already provides persistence. Subtract the channel.
3. **Separate cursor file for consumer offset.** A `.noodle/event-cursor.json` file tracking the schedule agent's read position. Rejected for now: adds a file and coordination complexity. The in-stream `schedule.completed` watermark is simpler and sufficient — the schedule agent already writes this event. If multiple consumers ever need independent cursors, add cursor files then.
4. **File-watch triggers (fsnotify per skill).** Rejected: couples skills to filesystem layout, doesn't handle non-file events.
5. **Inline handlers (hardcoded event→action in loop).** Rejected: defeats skills as the only extension point.

**Chosen:** NDJSON writer as infrastructure, mise brief as the bridge, schedule agent as the reactor. Maximum flexibility, minimum Go code, vision-aligned.

## Applicable Skills

- `go-best-practices` — Go patterns, concurrency, testing
- `testing` — fixture framework, test conventions

## Phases

1. [[archive/plans/66-event-trigger-system/phase-01-event-bus-types-and-ndjson-writer]]
2. [[archive/plans/66-event-trigger-system/phase-02-delete-audit-infrastructure]]
3. [[archive/plans/66-event-trigger-system/phase-03-migrate-snapshot-consumer]]
4. [[archive/plans/66-event-trigger-system/phase-04-emit-lifecycle-events-completion-and-merge]]
5. [[archive/plans/66-event-trigger-system/phase-05-emit-lifecycle-events-control-and-recovery]]
6. [[archive/plans/66-event-trigger-system/phase-06-surface-events-in-mise-brief]]
7. [[archive/plans/66-event-trigger-system/phase-07-external-event-cli]]
8. [[archive/plans/66-event-trigger-system/phase-08-teach-schedule-skill]]
9. [[archive/plans/66-event-trigger-system/phase-09-integration-tests]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

End-to-end: complete a cook, verify `stage.completed` appears in `loop-events.ndjson` and in the next mise brief's `recent_events`. Schedule agent sees it and can react.
