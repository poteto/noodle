---
id: 43
created: 2026-02-24
status: done
---

# Status File Split

## Context

Loop runtime state (`active`, `loop_state`, `autonomy`) lives in queue.json — the same file the prioritize agent writes. This mixes scheduling intent (agent-owned) with runtime status (noodle-owned), creating a potential write race and muddying the file's purpose.

The original plan also included deterministic self-healing (fixer interface, triage layer, simplified agent escalation). Those phases were scrapped when we decided to remove runtime repair entirely (plan 38 phase 3). What remains is the status file split — a clean separation of concerns.

## Scope

**In:**
- New `status.json` file for loop runtime state (`active`, `loop_state`, `autonomy`)
- Remove `Active`, `LoopState`, `Autonomy` fields from queue structs
- Update TUI snapshot, CLI commands, schema docs, and skill references

**Out:**
- Deterministic fixers, triage layer, agent escalation (deleted — repair is being removed in plan 38)
- TUI feed notifications for status changes

## Constraints

- The TUI uses poll-based refresh (tick → `loadSnapshot()`), not fsnotify — after the split, `loadSnapshot()` must read status.json in addition to queue.json. No new file watchers needed.
- `cmd_status.go` derives loop state from session metadata (`readSessionSummary`), not from queue.json — verify this still works and add status.json as a source if needed
- `cmd_debug.go` derives loop state from session metadata (`readDebugSessions`), not from queue.json — same consideration

## Applicable skills

- `go-best-practices` — Go patterns, testing conventions

## Phases

- [[archive/plans/43-deterministic-self-healing-and-status-split/phase-01-status-file-split]]
- [[archive/plans/43-deterministic-self-healing-and-status-split/phase-05-update-schema-and-skill-references]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```
