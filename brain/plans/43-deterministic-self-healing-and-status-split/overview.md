---
id: 43
created: 2026-02-24
status: ready
---

# Deterministic Self-Healing and Status Split

## Context

A vision audit identified two issues in noodle's current implementation:

1. **All repairs go through agents.** `runtime_repair.go` has one tool — spawn an agent. Malformed JSON, missing directories, stale locks — all trigger a 1-2 minute agent session when Go could fix them in milliseconds. The repair system also carries complexity it doesn't need: SHA1 fingerprinting, session adoption across restarts, dual Session/SessionID tracking paths.

2. **Loop state lives in queue.json.** The loop stamps `active`, `loop_state`, and `autonomy` into queue.json — the same file the prioritize agent writes. This mixes scheduling intent (agent-owned) with runtime status (noodle-owned), creating a potential write race and muddying the file's purpose.

This plan redesigns the repair flow from first principles and splits loop state into its own file.

## Scope

**In:**
- New `status.json` file for loop runtime state (`active`, `loop_state`, `autonomy`)
- Remove `Active`, `LoopState`, `Autonomy` fields from queue structs
- Fixer interface — registry of deterministic repair functions
- 5-6 fixer implementations (malformed queue/mise, missing dirs, stale sessions, orphaned worktrees, stale locks)
- Triage layer: run fixers first, escalate to agent only if unresolved
- Simplify agent escalation: drop fingerprinting, simplify adoption to lightweight reattach, keep attempt limits
- Update queue schema docs and prioritize skill to reference status.json
- Migrate existing fixture tests

**Out:**
- TUI feed notifications for deterministic repairs (todo #44 — deferred pending TUI feed redesign)
- Changes to what triggers repair (the 7 existing scopes stay the same)
- Repair prompt content changes (buildRuntimeRepairPrompt stays the same)
- Plan 38 overlap — that plan handles skill resolution resilience (non-fatal missing skills, oops fallback). This plan handles repair mechanics. The two are complementary: plan 38 ensures repair agents can always be spawned, this plan ensures most issues never need an agent.

## Constraints

- Existing fixture tests in `loop/testdata/runtime-repair-*` must be migrated — these test the current fingerprint/adoption flow
- `isProcessAlive()` already exists in `worktree/lock.go` — reuse it (move to a shared package or duplicate the small function)
- The TUI uses poll-based refresh (tick → `loadSnapshot()`), not fsnotify — after the split, `loadSnapshot()` must read status.json in addition to queue.json. No new file watchers needed.
- `cmd_status.go` derives loop state from session metadata (`readSessionSummary`), not from queue.json — verify this still works and add status.json as a source if needed
- `cmd_debug.go` derives loop state from session metadata (`readDebugSessions`), not from queue.json — same consideration

## Alternatives considered

**A. Add deterministic layer in front of existing runtime_repair.go** — Smallest change, keeps fingerprinting/adoption fully intact. Rejected: fingerprinting is overbuilt (SHA1 of scope+message+warnings when `scope|message` suffices). However, adoption can't be fully removed — if a repair agent is still running after a loop restart, we need to reattach to it rather than spawning a duplicate. The chosen approach simplifies adoption to a lightweight scan rather than deleting it entirely.

**B. Redesign repair flow holistically (chosen)** — Two tiers: Go fixes what's deterministic, agents fix what needs judgment. Simplify agent escalation by removing fingerprinting and adoption. The system is simpler despite adding functionality, because deterministic fixes are small testable functions and the agent path loses its overbuilt state tracking.

## Applicable skills

- `go-best-practices` — Go patterns, testing conventions
- `testing` — TDD workflow, fixture conventions

## Phases

- [[plans/43-deterministic-self-healing-and-status-split/phase-01-status-file-split]]
- [[plans/43-deterministic-self-healing-and-status-split/phase-02-fixer-interface-and-deterministic-fixers]]
- [[plans/43-deterministic-self-healing-and-status-split/phase-03-triage-layer-in-loop-cycle]]
- [[plans/43-deterministic-self-healing-and-status-split/phase-04-simplify-agent-escalation]]
- [[plans/43-deterministic-self-healing-and-status-split/phase-05-update-schema-and-skill-references]]
- [[plans/43-deterministic-self-healing-and-status-split/phase-06-update-and-migrate-tests]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```
