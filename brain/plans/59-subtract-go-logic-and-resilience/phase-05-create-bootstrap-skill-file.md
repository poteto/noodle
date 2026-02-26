Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 5: Create bootstrap skill file

Covers: #59 (skill creation)

## Goal

Create `.agents/skills/bootstrap/SKILL.md` containing the bootstrap prompt currently hardcoded in `builtin_bootstrap.go`. The prompt becomes an editable skill file that evolves without recompiling.

## Changes

- `.agents/skills/bootstrap/SKILL.md` — New file. Extract prompt content from `bootstrapPromptTemplate` in `loop/builtin_bootstrap.go`. Add frontmatter (name, description). This is NOT a task type skill (no `noodle:` block) — it's dispatched by the loop directly when the schedule skill is missing. Keep the `{{history_dirs}}` template variable — the loop will continue to substitute it before dispatch.

Use the `skill-creator` skill when making this change.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Skill writing — prompt quality matters |

## Verification

### Static
- File exists at `.agents/skills/bootstrap/SKILL.md`
- Frontmatter parses cleanly (`name: bootstrap`, `description: ...`)
- Prompt content matches current `bootstrapPromptTemplate` semantics (may improve wording)
- Contains `{{history_dirs}}` placeholder

### Runtime
- N/A — wiring happens in phase 6
