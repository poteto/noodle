---
id: 90
created: 2026-03-03
status: active
---

# Interactive Agent Sessions

Back to [[plans/index]]

## Context

Noodle's current execution model is fully automated: the scheduler creates orders, orders flow through staged pipelines (execute → quality → reflect), and humans interact only through steering and review. This works for well-defined coding tasks but is wrong for collaborative work — planning, exploration, design decisions — where a human needs to think alongside the agent in real time.

Interactive sessions are a new session category that lives outside the order system. No order ID, no stage pipeline, no quality/reflect lifecycle. The human starts a conversation, the agent responds, and they go back and forth until the work is done.

## Scope

**In scope:**
- New `chat` control action that spawns a provider session without order wrapping
- Multi-turn conversation: user sends messages, agent responds, repeat (steerable providers only)
- Interactive sessions appear in the AGENTS sidebar list with distinct styling
- Chat-oriented UI view with message-level user/assistant turn rendering
- Session launcher (prompt + optional name, default provider/model)
- Session history (conversations persist and replay on reconnect via existing event backfill)
- Crash recovery — interactive sessions survive loop restart

**Out of scope:**
- File/image attachments in chat input
- Unified spawn surface (orders + interactive from same place) — aspirational, not this plan
- Skill bundles for interactive sessions (use default system prompt)
- Cost budgets or turn limits for interactive sessions
- Worktree opt-in for interactive sessions — defer until there's evidence users need it
- Non-steerable provider multi-turn (Codex interactive sessions) — requires controller queue work first

## Constraints

- Reuse existing `processSession` / `sessionBase` infrastructure — interactive sessions are sessions, not a new primitive
- **Steerable providers only** for MVP — Claude sessions get live multi-turn via `AgentController.SendMessage()`. Non-steerable providers cannot do multi-turn without respawn, which breaks conversation continuity.
- Events flow through the same canonical pipeline — no new event types. Reuse `EventAction` with `tool: "user"` for chat messages (already used by steer). Differentiate rendering by session kind, not event type.
- Interactive sessions run on primary checkout (same as scheduler). This is a shared constraint — concurrent git mutations between scheduler + interactive sessions are possible but acceptable (same risk exists today between scheduler + bootstrap).
- Control ack protocol is append-only — `chat` ack cannot return a session ID synchronously. Use a correlation ID so the UI can discover the spawned session via snapshot.
- Session completion must route through the existing completion buffer — no direct goroutine mutation of loop maps (per serialize-shared-state-mutations).
- No backward compatibility concerns — this is net-new

## Alternatives Considered

**A. Reuse sessions, new spawn path (chosen):** Add `Kind` to session metadata, new control action bypasses order pipeline. Minimal new types, events/streaming for free. The session model wasn't designed for multi-turn, but the Claude controller already supports it.

**B. Separate "conversation" model:** New `Conversation` type distinct from `Session` with its own event stream and lifecycle. Cleaner separation but duplicates most of session infrastructure. Violates subtract-before-you-add.

**C. Interactive sessions as special orders:** Create a "chat" order type with no stages, spawn through existing pipeline. Zero new spawn infrastructure but the order model doesn't fit conceptually (no stages, no quality/reflect). Would pollute the order system with a non-order concept.

## Known Limitations (MVP)

- **Single-inflight message guard.** The Claude controller has a single-slot pending prompt queue. Rather than silently dropping messages, Phase 3 adds a guard that rejects a send when a message is already pending delivery, surfacing the constraint to the UI. A proper message queue in the controller is follow-up work.
- **Non-steerable providers are single-turn only.** The launcher restricts to steerable providers. Supporting Codex interactive sessions requires controller queue + respawn-with-context work.
- **Recovered sessions are stop-only.** After loop restart, interactive sessions are re-adopted and stoppable, but not steerable (the stdin pipe is lost). The UI shows recovered sessions in a "recovered" state. Re-establishing steerability after restart is follow-up work.

## Applicable Skills

- `go-best-practices` — Backend phases (session types, control actions, loop changes)
- `react-best-practices` — UI components (sidebar, chat view)
- `ts-best-practices` — TypeScript control command types, snapshot types
- `interaction-design` — Chat UI microinteractions, streaming feedback
- `frontend-design` — Chat panel design quality, distinct styling
- `codex` — Mechanical implementation phases (scaffold, types, wiring)

## Phases

1. [[plans/90-interactive-sessions/phase-01-kind-and-spawn]]
2. [[plans/90-interactive-sessions/phase-02-loop-tracking]]
3. [[plans/90-interactive-sessions/phase-03-multi-turn]]
4. [[plans/90-interactive-sessions/phase-04-snapshot]]
5. [[plans/90-interactive-sessions/phase-05-recovery]]
6. [[plans/90-interactive-sessions/phase-06-ui-sidebar-launcher]]
7. [[plans/90-interactive-sessions/phase-07-chat-view]]

## Verification

```bash
# Full suite
pnpm check

# Go tests (with race detector for concurrency safety)
go test -race ./...

# Go vet
go vet ./...

# Architecture lint
sh scripts/lint-arch.sh

# UI type check
cd ui && pnpm tsc --noEmit

# E2E: launcher → chat spawn → multi-turn → stop → cleanup chain
# E2E: loop restart with active interactive session → recovery → stop (not steer — recovered sessions are stop-only)
```
