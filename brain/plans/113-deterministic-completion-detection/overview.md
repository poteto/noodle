---
id: 113
created: 2026-03-03
status: active
---

# Plan 113: Deterministic Agent Completion Detection

Back to [[plans/index]]

## Context

Noodle's session completion detection uses a layered heuristic system that produces three failure modes: zombie sessions that linger after work is done, completed work misclassified as failed (or vice versa), and premature kills while agents are still writing files or committing.

The root architectural problem: `terminalStatus()` in `dispatcher/process_session.go` reads canonical events from disk **after** the process exits, scanning the log post-hoc to guess the outcome. This is backward — the system should track session state incrementally **during** processing, not reconstruct it from artifacts.

Additionally:
- Claude CLI never emits `EventComplete` — only `EventResult`. Codex emits `EventComplete` on clean exit but not on crash/timeout.
- Sprites sessions use exit-code-only detection (no event scanning), and don't write heartbeats, breaking monitor repair for remote sessions.
- The 2-second hard-coded shutdown deadline gives agents no meaningful time for cleanup.
- No verification that "completed" sessions actually produced their deliverables.

## Scope

**In scope:**
- Replace `terminalStatus()` heuristic with incremental state tracking
- Unified completion contract across processSession and spritesSession
- Configurable shutdown deadline with stage_yield-aware graceful shutdown
- Post-completion cleanup verification gate
- Evaluate whether `session_meta_repair.go` (zombie repair) can be removed

**Out of scope:**
- Agent steering/interrupt protocol changes (covered by sub-agent tracking #84)
- Provider CLI modifications (we work with what they emit)
- New canonical event types (we use the existing schema)

## Constraints

- Cross-platform (macOS/Windows/Linux) — signal handling differs per OS
- No backward compatibility shims — clean migration, delete legacy code
- Must handle providers that crash without emitting any completion event
- Sprites SDK mirrors `os/exec.Cmd` — same interface guarantees apply
- Event writes go through stamp processor (in-process) — no external file watching
- `stage_yield` is a session event (not canonical) — yield handling stays in the loop, not the tracker
- `Resolve()` must run AFTER stream processing completes — ordering is non-negotiable
- Zombie detection must remain claims-based — `Outcome()` is only populated after `Done()` closes

## Alternatives Considered

**A. Mandatory EventComplete from all providers:** Require adapters to synthesize `EventComplete` when the provider doesn't emit one (Claude). Rejected — this pushes completion semantics into the parser layer, conflating event normalization with lifecycle management. The parser's job is faithful translation, not lifecycle inference.

**B. Structured exit-code contract per provider:** Define exit codes (0=success, 1=failure, 2=partial) and enforce at the adapter layer. Rejected — exit codes are provider-controlled and unreliable. Claude exits non-zero for reasons unrelated to task completion (context overflow, tool errors). The exit code is a weak signal.

**C. Incremental completion tracker (chosen):** A state machine that runs alongside the stamp processor, consuming the same event stream and tracking session lifecycle in real time. When the process exits, the outcome is already known — no disk scanning needed. This is the foundational-thinking approach: get the data structure (session state machine) right, and downstream logic becomes obvious.

## Applicable Skills

- `go-best-practices` — lifecycle management, concurrency patterns
- `testing` — test-driven verification for each phase

## Phases

1. [[plans/113-deterministic-completion-detection/phase-01-session-outcome-type]]
2. [[plans/113-deterministic-completion-detection/phase-02-completion-tracker]]
3. [[plans/113-deterministic-completion-detection/phase-03-wire-tracker-into-session-base]]
4. [[plans/113-deterministic-completion-detection/phase-04-replace-terminal-status]]
5. [[plans/113-deterministic-completion-detection/phase-05-sprites-session-alignment]]
6. [[plans/113-deterministic-completion-detection/phase-06-sprites-heartbeat]]
7. [[plans/113-deterministic-completion-detection/phase-07-graceful-shutdown]]
8. [[plans/113-deterministic-completion-detection/phase-08-cleanup-verification]]
9. [[plans/113-deterministic-completion-detection/phase-09-evaluate-monitor-repair]]
10. [[plans/113-deterministic-completion-detection/phase-10-delete-legacy]]

## Verification

- `go test ./dispatcher/... ./loop/... ./monitor/... ./stamp/...`
- `go vet ./...`
- `pnpm check` (full suite)
- Manual: spawn sessions with each provider, verify correct status on clean exit, crash, timeout, and SIGKILL
