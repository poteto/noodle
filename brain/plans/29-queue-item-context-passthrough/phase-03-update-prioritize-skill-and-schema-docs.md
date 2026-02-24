Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 3: Update prioritize skill and schema docs

## Goal

Document the `extra_prompt` field so the prioritize agent knows it exists and how to use it. The schemadoc FieldDoc entry was already added in phase 1 to keep `noodle schema queue` valid from the start.

## Changes

### Update `.agents/skills/prioritize/SKILL.md`

Add guidance in the **Output** section (or a new subsection near it) explaining the `extra_prompt` field:

- Purpose: supplemental instructions about *how* to do the task, distinct from `prompt` (what to do) and `rationale` (why it's scheduled)
- Use cases: relay failure context from `recent_history`, flag dependencies, suggest approach constraints
- Keep it concise (~1000 chars max, silently truncated if exceeded)
- Leave empty when there's nothing extra to say — don't fill it for the sake of filling it

## Data structures

No new types. Updates documentation for the existing `ExtraPrompt` field.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Documentation-only change to one file with clear spec |

## Verification

### Static
```bash
go test ./... && go vet ./...
noodle schema queue
```

### Runtime
- Run `noodle schema queue`, verify `extra_prompt` appears in the output with its description
- Run the prioritize agent, verify it can produce `extra_prompt` in `queue-next.json` (manual or integration test)
