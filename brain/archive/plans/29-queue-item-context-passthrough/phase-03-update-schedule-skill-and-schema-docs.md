Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 3: Update schedule skill and schema docs

## Goal

Document the `extra_prompt` field so the schedule agent knows it exists and how to use it. The schemadoc FieldDoc entry was already added in phase 1 to keep `noodle schema orders` valid from the start.

## Changes

### Update `.agents/skills/schedule/SKILL.md`

Add guidance in the **Output** section (or a new subsection near it) explaining the `extra_prompt` field on stages:

- Purpose: supplemental instructions about *how* to do the task, distinct from `prompt` (what to do) and `rationale` (why it's scheduled)
- Use cases: relay failure context from `recent_history`, flag dependencies, suggest approach constraints
- Keep it concise (~1000 chars max, silently truncated if exceeded)
- Leave empty when there's nothing extra to say — don't fill it for the sake of filling it
- Field lives on each stage in `orders-next.json`, not at the order level

## Data structures

No new types. Updates documentation for the existing `ExtraPrompt` field.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Documentation-only change to one file with clear spec |

## Verification

### Static
```bash
go test ./... && go vet ./...
noodle schema orders
```

### Runtime
- Run `noodle schema orders`, verify `extra_prompt` appears in the output with its description
- Run the schedule agent, verify it can produce `extra_prompt` in `orders-next.json` (manual or integration test)
