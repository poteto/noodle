Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 11: Utility Skills — Debugging + Plan Skill Update

## Goal

Light amendments to the debugging utility skill, and update the plan skill for native commands. Neither is a task type — no `noodle:` frontmatter.

## Debugging

### Current State

- `.agents/skills/debugging/SKILL.md` — well-structured root-cause methodology, already principle-grounded. Missing Noodle-specific repair context.

### Changes

- Rewrite `.agents/skills/debugging/SKILL.md` — **use the `skill-creator` skill**
- Add Noodle-specific repair context: common failure modes (missing skills, stale queue, config errors, tmux issues)
- Add `.noodle/` state file inspection as diagnostic step
- Add brain-update step: capture novel gotchas
- Keep existing root-cause methodology — it's well-written
- No `noodle:` frontmatter — this is a utility skill invoked by oops (user-project fixes), repair (Noodle-internal fixes), and execute (diagnostic steps)

### Principles

- [[principles/observe-directly]] — for repair: check `.noodle/` state files directly
- [[principles/never-block-on-the-human]] — repair sessions must be fully autonomous

## Plan Skill

### Current State

- `.agents/skills/plan/SKILL.md` — references stale `go -C noodle run .` commands

### Changes

- Update `.agents/skills/plan/SKILL.md` — **use the `skill-creator` skill**
- Replace `go -C noodle run . plan create/phase` with native `noodle plan create/phase-add`
- Replace `go -C noodle run . todo add/edit` with configured backlog adapter commands
- Add Routing section to phase file template (provider/model/rationale)
- Add routing guidance table:

| Phase type | Provider | Model | When |
|------------|----------|-------|------|
| Implementation | `codex` | `gpt-5.3-codex` | Coding against a clear spec |
| Architecture / judgment | `claude` | `claude-opus-4-6` | Design decisions, complex refactoring |
| Skill creation | `claude` | `claude-opus-4-6` | Writing skills, brain notes |
| Scaffolding | `codex` | `gpt-5.3-codex` | Boilerplate, mechanical transforms |

- Update backlog item link-back via configured adapter after plan creation
- No `noodle:` frontmatter — planning is user-invoked, not Noodle-scheduled

## Verification

- Static: Debugging SKILL.md has Noodle repair context, `.noodle/` diagnostic step
- Static: Plan skill uses `noodle plan create/phase-add` (native commands)
- Static: Plan skill uses backlog adapter for todo operations
- Static: Phase file template includes Routing section
- No references to `go -C noodle run . plan` or `go -C noodle run . todo`
- Spawn a plan session: verify phase files include Routing sections
