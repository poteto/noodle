Back to [[plans/20-onboarding/overview]]

# Phase 4 — Concept Doc: Skills

## Goal

Write the skills concept page for the docs site. Cover what skills are, how they're discovered, the file format, and how to create one. This is the most important concept doc — skills are the only extension point.

## Changes

- **`docs/concepts/skills.md`** — New page covering:
  - What a skill is (directory with SKILL.md)
  - Frontmatter fields (name, description, model, schedule)
  - The `schedule` field — how it turns a skill into a task type
  - Skill path discovery: `.agents/skills` (default), `.claude/skills`, custom paths via `[skills.paths]` in `.noodle.toml`
  - First-match-wins resolution order
  - Task-type skills (with `schedule` field) vs general skills (invoked directly)
  - Creating your first skill (minimal example)
  - Reference files (`references/` subdirectory)

## Data structures

- `skill.Frontmatter` — document the fields from `skill/frontmatter.go`

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Needs accurate technical writing with good examples |

Apply `unslop` skill.

## Verification

### Static
- Page builds in VitePress without errors
- All code examples are syntactically valid
- Frontmatter field documentation matches `skill/frontmatter.go`

### Runtime
- A new user can follow the "creating your first skill" section and it works
- Skill path documentation matches actual resolver behavior (test with `noodle skills list`)
