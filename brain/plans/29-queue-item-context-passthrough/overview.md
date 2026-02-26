---
id: 29
created: 2026-02-24
status: ready
---

# Backlog-Only Scheduling with Context Passthrough

## Context

Two problems, one solution:

**1. No context passthrough.** The schedule agent can't pass supplemental instructions to cooks. If it knows from `recent_history` that a previous attempt failed because tests weren't run, it has no way to tell the next cook "run tests this time." The `prompt` field is for *what* to do, `rationale` is for *why* it was scheduled — neither is for *how*.

**2. Brittle plan integration.** Noodle's Go code cross-references backlog IDs with plan IDs via `needs_scheduling` to decide what's ready. This means plans are mandatory for scheduling, the plan format is baked into Go, and the coupling between backlog/plans/mise is fragile. The plan step is also too opinionated for most users.

The fix: add an `extra_prompt` field for context passthrough (phases 1-4), then simplify to backlog-only scheduling where the backlog adapter is the single integration point and plans are optional context surfaced through backlog items (phases 5-7).

## Scope

**In (phases 1-4 — context passthrough):**
- New `ExtraPrompt string` field on QueueItem
- Inject `Scheduling context:` section in `buildCookPrompt()`
- Soft size guardrail: truncate to ~1000 chars
- Update schedule skill and schema docs
- Tests

**In (phases 5-7 — backlog-only scheduling):**
- Remove native plan reader from mise builder (`plan.ReadAll()`, `schedulablePlanIDs()`, `needs_scheduling`) — atomic with skill rewrite
- Backlog adapter protocol gains optional `plan` path field on items (new field, not migration)
- Schedule skill reads plan files from backlog items and injects phase context via `extra_prompt`
- Update loop idle gate, queue validator, bootstrap scheduler instructions, schemadoc
- First-run bootstrap: no adapter → prompt user to create one with context about their work source
- Delete plans-sync from adapter; `plan` package stays (used by `noodle plan` CLI)

**Out:**
- Structured context fields (tags, key-value pairs) — free-form string is sufficient
- Hard validation rejection (truncation is enough)
- Changes to the `rationale` field's semantics
- Changing Noodle's own `brain/plans/` format — it stays, just becomes the adapter's concern
- Deleting the plan skill (`.claude/skills/plan/`) — it's an interactive Claude skill, not a Noodle adapter

## Constraints

- `extra_prompt` is free-form string, not structured. The scheduler writes natural language.
- `buildCookPrompt()` must not emit blank lines when `extra_prompt` is empty.
- Truncation happens at queue read/normalize time.
- Phases 1-4 ship independently — they add value even without the backlog overhaul.
- Phases 5-7 are sequential. Phase 5 is internally atomic (plan reader removal + skill rewrite ship together). Phase 6 (bootstrap) should ship soon after — without it, first-run projects with no adapter can idle with no path forward.
- The loop idle gate (`loop/loop.go`) currently idles when `plans` and `needs_scheduling` are empty. After removal, it must idle on empty backlog instead.
- Queue validation (`internal/queuex/queue.go`) enforces execute-ID membership against `schedulablePlanIDs` — this must be migrated.
- Bootstrap scheduler instructions (`loop/builtin_bootstrap.go`) hardcode `needs_scheduling` — must be updated.
- The backlog adapter is the *only* integration point after this. A user who uses Linear writes a Linear adapter. A user who uses markdown writes a markdown adapter. Noodle core doesn't know or care.

## Applicable skills

- `go-best-practices` — Go patterns, testing conventions
- `testing` — TDD workflow

## Phases

**Context passthrough (ships first):**
- [[plans/29-queue-item-context-passthrough/phase-01-add-extra-prompt-field-to-queueitem]]
- [[plans/29-queue-item-context-passthrough/phase-02-inject-extra-prompt-into-cook-prompt]]
- [[plans/29-queue-item-context-passthrough/phase-03-update-schedule-skill-and-schema-docs]]
- [[plans/29-queue-item-context-passthrough/phase-04-tests]]

**Backlog-only scheduling (depends on phases 1-4):**
- [[plans/29-queue-item-context-passthrough/phase-05-atomic-plan-removal-and-skill-rewrite]] — atomic: remove plan reader + add backlog plan field + rewrite schedule skill
- [[plans/29-queue-item-context-passthrough/phase-06-first-run-backlog-adapter-bootstrap]] — first-run backlog adapter bootstrap
- [[plans/29-queue-item-context-passthrough/phase-07-delete-plans-sync-and-clean-up]] — delete plans-sync and clean up

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

After phases 1-4: schedule agent can write `extra_prompt`, cooks receive it as `Scheduling context:`.

After phases 5-7: `mise.json` has no `plans` or `needs_scheduling`, backlog items optionally carry plan paths, schedule skill reads plans itself, `noodle start` without a backlog adapter prompts to create one, `noodle plan` CLI still works.
