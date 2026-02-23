Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 3: Context Injection — Preamble + Multi-Skill Loading

## Goal

Bridge the lean Go core and smart skills with proper context injection. Every cook session gets a state model map. Task types that need both a domain skill and a methodology skill get both loaded.

## Current State

- Cook sessions get `"Work backlog item <id>"` + optional rationale — no context about `.noodle/` state
- `spawner/skill_bundle.go` loads a single skill's SKILL.md + references/
- `TaskType` struct has a single `Skill` field
- Agents must discover `.noodle/` files by filesystem exploration

## Changes

### Noodle context preamble

Add a lean document injected into every cook session's system prompt that maps the state model:

- `.noodle/mise.json` — system state snapshot (backlog, plans, sessions, history)
- `.noodle/queue.json` — work queue the session was scheduled from
- `.noodle/tickets.json` — claimed resources and concurrency locks
- `.noodle/debates/<task-id>/` — debate state per task
- `.noodle/quality/<session-id>/` — quality verdicts
- `brain/plans/` — plan files for phase details
- `brain/todos.md` — backlog items

The preamble says "here's what exists and why" — not schemas. Schemas live in each skill's `references/`.

### Multi-skill loading

The execute task type needs both the adapter-configured skill (what to work on) and the execute methodology skill (how to work). Update:

- Add `Skills []string` field (or `MethodologySkill` field) to the task type struct
- Update `spawner/skill_bundle.go:loadSkillBundle()` to compose prompts from multiple skills
- Domain skill first, methodology skill second

### Skill-specific schema references

Each task-type skill that reads/writes `.noodle/` state must include schema docs in `references/`:

| Skill | References |
|-------|------------|
| prioritize | `references/mise-schema.md`, `references/queue-schema.md` |
| quality | `references/verdict-schema.md` |
| debate | `references/debate-state-schema.md` |

## Data Structures

- Preamble: static markdown template, injected by spawner
- Skill bundle: ordered list of SKILL.md contents + merged references/

## Verification

- `make ci` passes
- Unit tests for multi-skill composition: ordering, truncation at provider limits, single-skill fallback
- Unit tests for preamble injection: preamble is prepended, content matches state model map
- Cook session system prompt includes preamble + skill content
- Execute task loads both domain and methodology skills
