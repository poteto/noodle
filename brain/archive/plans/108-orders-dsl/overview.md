---
id: 108
created: 2026-03-02
status: active
---

# Compact Wire Format for orders-next.json

## Context

The scheduler LLM emits `orders-next.json`, which the loop promotes into `orders.json`. Every stage currently requires 6 fields (`task_key`, `skill`, `provider`, `model`, `runtime`, `status`) — two of which are redundant (`skill` duplicates `task_key`, `status` is always `"pending"`) and two use unnecessarily verbose names. Research in [[plans/108-orders-dsl/sketch]] explored format alternatives.

## Scope

**In scope:**
- `do` replaces `task_key` (shorter, reads as a verb)
- `with` replaces `provider` (shorter, reads naturally with `do`)
- Drop `status` from scheduler output — loop sets stages to `pending`, orders to `active`
- Drop `skill` from scheduler output — resolved from `task_key` via registry
- Expansion layer at the promotion boundary (compact → internal types)
- Schema docs, scheduler skill, and test fixture updates

**Out of scope:**
- `parallel` arrays replacing `group` integers (future, if needed)
- Template references (future, if needed)
- Changing internal `orderx.Stage`/`orderx.Order` types — they stay as-is
- Dual format support — hard cut

## Constraints

- **Boundary discipline:** Expansion happens at the promotion boundary (`consumeOrdersNext`). Internal code sees the existing types unchanged.
- **Idempotent promotion:** Crash-safe read→merge→write→delete flow unchanged.
- **No backward compat:** Scheduler and loop switch together.
- **Schema from types:** `schemadoc` generates the schema prompt from Go types via reflection. New compact types must be reflected.

## Wire Format Design

### Before (current)

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature",
      "status": "active",
      "stages": [
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "sprites", "status": "pending"},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "reflect", "skill": "reflect", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

### After (compact)

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature",
      "stages": [
        {"do": "execute", "with": "codex", "model": "gpt-5.3-codex", "runtime": "sprites"},
        {"do": "quality", "with": "claude", "model": "claude-opus-4-6"},
        {"do": "reflect", "with": "claude", "model": "claude-opus-4-6"}
      ]
    }
  ]
}
```

### Field semantics

| Wire field | Expands to | Notes |
|------------|-----------|-------|
| `do` | `task_key` (+ `skill` resolved from registry) | Optional if `prompt` is set. At least one of `do` or `prompt` required. |
| `with` | `provider` | Required |
| `model` | `model` | Required |
| `runtime` | `runtime` | Optional. Loop defaults to `"process"` at dispatch. |
| `prompt` | `prompt` | Optional if `do` is set. At least one of `do` or `prompt` required. |
| `extra_prompt` | `extra_prompt` | Optional. Unchanged. |
| `extra` | `extra` | Optional. Unchanged. |
| `group` | `group` | Optional. Unchanged. |

### Ad-hoc stages (no task key)

```json
{
  "id": "investigate-flaky-ci",
  "title": "debug flaky test in loop package",
  "stages": [
    {"with": "claude", "model": "claude-opus-4-6", "prompt": "Investigate and fix the flaky TestCyclePromotion test in loop/"}
  ]
}
```

A stage without `do` is ad-hoc — it has a `prompt` but no registered task type. A stage can have `do` only, `prompt` only, or both. Validation rejects stages with neither.

**Dropped from wire format:**
- `status` on stages — loop sets all to `pending`
- `status` on orders — loop sets all to `active`
- `skill` — resolved from `task_key` via registry at dispatch time

## Applicable Skills

- `go-best-practices` — for all implementation phases
- `testing` — for test-heavy phases
- `skill-creator` — for phase 3 (scheduler skill update)

## Phases

1. [[plans/108-orders-dsl/phase-01-compact-types]]
2. [[plans/108-orders-dsl/phase-02-promotion]]
3. [[plans/108-orders-dsl/phase-03-schema-and-skill]]

## Verification

After all phases: `pnpm check` (full suite), `go test ./internal/orderx/... ./loop/... ./internal/schemadoc/... ./internal/snapshot/...`
