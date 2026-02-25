---
id: 29
created: 2026-02-24
status: ready
---

# Queue Item Context Passthrough

## Context

The schedule agent schedules queue items with a `prompt` (full task text) and a `rationale` (why the item was scheduled). Neither field gives the scheduler a way to pass supplemental instructions about *how* a task should be done. For example, the scheduler might observe from `recent_history` that a previous cook failed because it forgot to run tests, or that a particular plan phase needs a careful approach due to upstream dependencies. Today it can only encode this guidance by bloating the `prompt` field or shoehorning advice into `rationale`, which muddies both fields' purpose.

A new `extra_prompt` field on QueueItem gives the scheduler a dedicated channel for scheduling context. It gets injected into the cook's prompt under a `Scheduling context:` prefix, keeping it visually distinct from the task description and rationale.

## Scope

**In:**
- New `ExtraPrompt string` field on QueueItem (both `loop/types.go` and `internal/queuex/queue.go`)
- Inject `Scheduling context:` section in `buildCookPrompt()` — omitted when field is empty
- Soft size guardrail: silently truncate to ~1000 characters if exceeded
- Update schedule skill (`SKILL.md`) to document the field and its intended use
- Update queue schema docs in `internal/schemadoc/specs.go`
- Tests for prompt construction with/without extra_prompt, and truncation behavior

**Out:**
- Structured context fields (tags, key-value pairs) — free-form string is sufficient
- Hard validation rejection (truncation is enough; don't fail the queue)
- Changes to how the schedule agent decides *what* to write in the field (that's agent judgment)
- Changes to the `rationale` field's semantics

## Constraints

- Field is free-form string, not structured. The scheduler writes natural language.
- `buildCookPrompt()` must not emit blank lines or a heading when `extra_prompt` is empty — existing tests verify no double blank lines.
- Truncation happens at queue read/normalize time, not at prompt construction time — keep `buildCookPrompt()` simple.
- The `extra_prompt` JSON key uses snake_case to match `task_key`, `rationale`, etc.

## Applicable skills

- `go-best-practices` — Go patterns, testing conventions
- `testing` — TDD workflow

## Phases

- [[plans/29-queue-item-context-passthrough/phase-01-add-extra-prompt-field-to-queueitem]]
- [[plans/29-queue-item-context-passthrough/phase-02-inject-extra-prompt-into-cook-prompt]]
- [[plans/29-queue-item-context-passthrough/phase-03-update-schedule-skill-and-schema-docs]]
- [[plans/29-queue-item-context-passthrough/phase-04-tests]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```
