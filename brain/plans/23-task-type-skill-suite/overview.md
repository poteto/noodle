---
id: 23
created: 2026-02-22
status: draft
---

# Task-Type Skill Suite + First-Class Planning

Combines todo #23 (task-type skills) and todo #22 (opinionated planning).

## Context

Noodle's task type registry defines 12 task types, each mapping to a skill that a cook session invokes. Several skills are missing entirely (quality, oops), several exist but weren't designed for cook sessions or grounded in engineering principles (prioritize, reflect, meditate, debugging), and 5 old role-based skills (CEO, CTO, Director, Manager, Operator) contain valuable patterns that should be extracted before deletion.

Additionally, planning is currently implemented as a user-configurable adapter — but it should be opinionated and first-class. Plans always live in `brain/plans/` with a Noodle-owned format. The adapter indirection adds complexity without value.

Several existing skills also reference CLI commands that no longer exist (`noodle todo`, `noodle plan`).

## Scope

**In scope:**
- Create or rewrite 8 skills in `.agents/skills/` — 7 task-type skills + 1 utility skill (debugging)
- Extract valuable patterns from old role-based skills (CEO, CTO, Director, Manager, Operator)
- Ground each skill in engineering principles from `brain/principles/`
- Design for cook sessions — Noodle handles autonomy via spawn flags, skills don't need to manage it
- Make planning native: remove plan adapter, add minimal Go reader for `brain/plans/` metadata, add `noodle plan` CLI commands
- Add model routing recommendations to plan phase files
- Plan skill updates backlog item to link back to created plan
- Add interactive TUI planning session (chef chats with sous-chef)
- Add Noodle context preamble to cook session spawner (state model map for agents)
- Each skill includes relevant `.noodle/` schemas in `references/` directory
- Fix stale CLI references across all existing skills
- Delete old role-based skills after extraction

**Out of scope:**
- Review skill — the Chef (human) does review via the TUI, not an LLM agent
- Verify skill — the execute agent verifies its own work (tests, lint, plan completeness check) before committing
- Backlog adapter changes — backlogs stay configurable (GitHub Issues, Linear, etc.)
- Bootstrap skill updates
- Interactive-only skills that don't map to task types (commit, codex, skill-creator, etc.)

## Constraints

- **Lean core, smart skills.** Noodle's Go core is a thin orchestration layer: process lifecycle, concurrency, file I/O, and data assembly. All scheduling intelligence, quality judgment, and task semantics live in skills. The Go core surfaces data (mise brief, plan metadata, session history); skills read that data and make decisions. This keeps the core extensible — users customize behavior by writing skills, not by modifying Go code.
- **Everything is a file.** Skills, brain notes, plans, `.noodle/` state — all are files the agent reads directly. This makes agents powerful (full filesystem access) and the tool extensible (users modify files, not config APIs).
- **Context injection bridges core and skills.** The Go core surfaces data as files, but agents need to know those files exist and what they mean. Two layers handle this: (1) a **Noodle context preamble** injected by the spawner into every cook session — a lean map of `.noodle/` state files and their purpose, and (2) **skill-specific schemas** in each skill's `references/` directory documenting the exact data that skill reads and writes. The preamble says "here's what exists"; the skill references say "here's how to use it."
- All skills live in `.agents/skills/`. No `skills/` stubs directory — users who want to scaffold skills can reference the Noodle repo directly.
- Cook sessions are autonomous by default — Noodle passes flags to disable interactive prompts (e.g. `--no-input` for Claude, equivalent for Codex). Skills don't need to handle this themselves.
- Skills should be lean — guard the context window. Every line must earn its place in a cook session's system prompt.
- Use the `skill-creator` skill when writing each skill to ensure quality and consistency.

## Old Skill Extraction Map

Patterns worth preserving from the old role-based skills:

| Source | Pattern | Target |
|--------|---------|--------|
| CEO | Foundation-before-feature ordering | prioritize |
| CEO | Cheapest execution mode that finishes safely | prioritize |
| CEO | Explicit rationale for every scheduling decision | prioritize |
| CEO | Fresh context each cycle — no long-lived drift | prioritize |
| CEO | Work around blockers, don't idle | prioritize |
| CTO | Evidence-first quality governance | quality |
| CTO | Lint-arch as first-class audit evidence | quality |
| CTO | Principle-anchored findings with citations | quality |
| CTO | Advocate, don't block — preserve momentum | quality |
| Manager | "Claims are promises, not proof" — verify artifacts | quality, execute |
| Manager | git diff --stat ALL files, not just claimed | quality, execute |
| Operator | Decompose → Implement → Verify → Commit flow | execute, oops |
| Operator | Worktree isolation, lint-before-commit | execute, oops |
| Operator | Brain update on fix — capture novel gotchas | debugging |
| Manager | Parallel by default — sub-agents for independent work | execute |
| Manager | Minimal-context workers — front-load context | execute |

## Principle-to-Skill Mapping

| Principle | Skills |
|-----------|--------|
| verify-runtime | quality, debugging, execute |
| trust-the-output-not-the-report | quality, execute |
| fix-root-causes | debugging, oops |
| observe-directly | debugging, oops |
| suspect-state-before-code | debugging, oops |
| encode-lessons-in-structure | reflect, meditate |
| cost-aware-delegation | prioritize, execute |
| foundational-thinking | prioritize |
| guard-the-context-window | all |
| never-block-on-the-human | all |
| subtract-before-you-add | meditate, prioritize |
| exhaust-the-design-space | debate |
| boundary-discipline | execute |
| outcome-oriented-execution | quality, execute, debate |
| redesign-from-first-principles | meditate |

## Phases

### Task-type skills

1. [[plans/23-task-type-skill-suite/phase-01-prioritize]] — Rewrite queue scheduler with CEO scheduling judgment
2. [[plans/23-task-type-skill-suite/phase-02-quality]] — Post-cook quality gate
3. [[plans/23-task-type-skill-suite/phase-03-reflect]] — Cook-session-first learning from mistakes and corrections
4. [[plans/23-task-type-skill-suite/phase-04-meditate]] — Cook-session-first brain cleanup and principle extraction
5. [[plans/23-task-type-skill-suite/phase-05-oops]] — User-project infrastructure fix skill
6. [[plans/23-task-type-skill-suite/phase-06-debugging]] — Root-cause diagnosis utility (invoked by oops/execute/repair, not a task type)
7. [[plans/23-task-type-skill-suite/phase-07-debate]] — Structured debate with per-task state in `.noodle/debates/<task-id>/`
8. [[plans/23-task-type-skill-suite/phase-08-execute]] — Implementation methodology skill (worktrees, delegation, verification)

### First-class planning

9. [[plans/23-task-type-skill-suite/phase-09-native-planning]] — Remove plan adapter, minimal Go reader + CLI commands
10. [[plans/23-task-type-skill-suite/phase-10-plan-skill]] — Update plan skill for native commands + model routing + backlog link-back
11. [[plans/23-task-type-skill-suite/phase-11-tui-planning]] — Interactive TUI planning session (chef chats with sous-chef)

### Cleanup

12. [[plans/23-task-type-skill-suite/phase-12-stale-references]] — Fix stale CLI references across remaining skills
13. [[plans/23-task-type-skill-suite/phase-13-cleanup]] — Delete old skills, rename sous-chef→prioritize, Go code updates

## Verification

- Each skill SKILL.md has: frontmatter, purpose, principles, contract, process, verification
- Skill resolver finds each skill: `go test ./skill/...`
- Old role-based skills (CEO, CTO, Director, Manager, Operator) are deleted
- No remaining references to `sous-chef` in Go code or config
- Verify task type removed from `internal/taskreg/registry.go`
- `noodle plan create/done/phase-add` commands work
- No `[adapters.plans]` in config
- Mise brief includes plan metadata (prioritize skill computes associations)
- Plan skill updates backlog item to link back to created plan
- Plan phases include Routing sections with provider/model
- Execute skill is loaded alongside adapter-configured skill for execute task type
- Noodle context preamble is injected into all cook sessions (agents can locate `.noodle/` state files)
- Skills that read/write `.noodle/` state include schema docs in `references/` (prioritize, quality, debate)
- TUI planning session produces valid plans
- No remaining references to stale CLI commands
- `go vet ./...` and `go test ./...` pass
