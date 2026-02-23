Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 8: Execute — Implementation Methodology Skill

## Goal

Create `.agents/skills/execute/SKILL.md` — the skill that teaches cook sessions how to do implementation work. This is the most-spawned task type and currently has no methodology skill. The cook prompt provides _what_ to do (backlog item + plan phase); this skill provides _how_ to do it.

## Current State

- The execute task type maps to the adapter-configured skill (`todo`), which teaches backlog management — not execution methodology
- Cook prompt is just `"Work backlog item <id>"` with rationale context
- No skill provides: worktree discipline, sub-agent delegation, lint-before-commit, verification, commit conventions
- The Operator skill had this methodology but it's being deleted

## Patterns to Incorporate

From **Operator**:
- **Decompose → Implement → Verify → Commit** — structured execution flow
- **Worktree isolation** — work in a worktree, not the primary checkout
- **Lint-before-commit** — run linting and fix ALL issues before committing
- **Task tracking** — create tasks after decomposition, mark in_progress/completed at each step

From **Manager** (for phases that need delegation):
- **Parallel by default** — when a phase touches independent areas, use sub-agents
- **Minimal-context workers** — front-load context to avoid rediscovery costs
- **Verify artifacts, not reports** — git diff --stat ALL files after sub-agents complete

## Principles

- [[principles/verify-runtime]] — verify your own work before calling it done
- [[principles/trust-the-output-not-the-report]] — inspect artifacts (git diff --stat ALL files), not sub-agent summaries
- [[principles/cost-aware-delegation]] — use sub-agents when beneficial, self-execute when simpler
- [[principles/guard-the-context-window]] — delegate large reads to sub-agents
- [[principles/boundary-discipline]] — only change what the phase specifies, flag anything else
- [[principles/outcome-oriented-execution]] — optimize for the end state, not smooth intermediate steps

## Changes

- Create `.agents/skills/execute/SKILL.md` — **use the `skill-creator` skill**
- This skill is cook-session-first — autonomous execution is the only mode
- Define the execution flow:
  1. Read the plan phase (if one exists) — it has goal, changes, data structures, verification, routing
  2. Decompose into sub-tasks if the phase is large enough
  3. Work in a worktree (invoke worktree skill)
  4. Implement, using sub-agents for independent sub-tasks
  5. Verify: run tests, lint, check git diff against expected changes, and compare against plan phase requirements (confirm all items in Changes and Verification sections are addressed)
  6. Commit with conventional message
- Include delegation heuristics: when to self-execute vs spawn sub-agents
- Include scope discipline: only change what the phase specifies, flag anything else
- The execute skill provides _methodology_ (how to work) alongside the adapter-configured skill (which provides _what_ to work on). The current `TaskType` struct has a single `Skill` field — this phase must add a `Skills []string` field (or `MethodologySkill` field) to support loading both the adapter skill and the execute methodology skill for the same task type. Update `spawner/skill_bundle.go:loadSkillBundle()` to compose prompts from multiple skills.

### Noodle context preamble

The spawner currently injects no context about Noodle's state model into cook sessions. The agent must discover `.noodle/` files by exploring the filesystem. Add a **Noodle context preamble** — a lean document injected into every cook session's system prompt that maps the state model:

- `.noodle/mise.json` — current system state snapshot (backlog, plans, sessions, history)
- `.noodle/queue.json` — work queue the session was scheduled from
- `.noodle/tickets.json` — claimed resources and concurrency locks
- `.noodle/debates/<task-id>/` — debate state per task
- `.noodle/quality/<session-id>/` — quality verdicts
- `brain/plans/` — plan files the agent can read for phase details
- `brain/todos.md` — backlog items

The preamble is a map, not a schema — it says "here's what exists and why" so the agent knows where to look. Detailed schemas live in each skill's `references/` directory. This is a small Go change in the spawner, not a new package.

### Skill-specific schema references

Each task-type skill must include `references/` files documenting the schemas it reads and writes:

| Skill | References |
|-------|------------|
| prioritize | `mise-schema.md`, `queue-schema.md` |
| quality | `verdict-schema.md` |
| debate | `debate-state-schema.md` |
| execute | (none — reads plan files directly, no Noodle-specific schema) |
| oops | (none — reads error context from cook prompt) |
| reflect | (none — reads/writes brain files) |
| meditate | (none — reads/writes brain files) |

This ensures each agent gets exactly the schema documentation it needs — no more, no less.

## Data Structures

- Input: cook prompt (backlog item ID + rationale) + plan phase file (if exists)
- Output: committed code changes in the worktree

## Verification

- Static: SKILL.md has frontmatter, principles, execution flow, delegation heuristics, scope discipline
- Static: Noodle context preamble is injected into cook session system prompts
- Static: Skills that read/write `.noodle/` state include relevant schema docs in `references/`
- Runtime: Spawn an execute cook session for a plan phase. Confirm:
  - Agent reads the plan phase file
  - Agent works in a worktree
  - Agent runs tests and lint before committing
  - Agent checks all plan phase requirements are addressed before committing
  - Commit message references the backlog item
  - Only files listed in the phase's Changes section are modified (scope discipline)
  - Agent can locate `.noodle/` state files without filesystem exploration (preamble works)
