Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 6: Execute — Implementation Methodology

## Goal

Create `.agents/skills/execute/SKILL.md` — the skill that teaches cook sessions how to do implementation work. The cook prompt provides _what_ to do; this skill provides _how_.

## Current State

- The execute task type maps to the adapter-configured skill (`todo`), which teaches backlog management — not execution methodology
- No skill provides: worktree discipline, sub-agent delegation, lint-before-commit, verification, commit conventions

## Patterns to Incorporate

From **Operator**: Decompose → Implement → Verify → Commit, worktree isolation, lint-before-commit, task tracking.
From **Manager**: parallel by default, minimal-context workers, verify artifacts not reports.

## Principles

- [[principles/prove-it-works]] — verify your own work before calling it done
- [[principles/trust-the-output-not-the-report]] — inspect artifacts, not sub-agent summaries
- [[principles/cost-aware-delegation]] — use sub-agents when beneficial, self-execute when simpler
- [[principles/guard-the-context-window]] — delegate large reads to sub-agents
- [[principles/boundary-discipline]] — only change what the phase specifies
- [[principles/outcome-oriented-execution]] — optimize for the end state

## Changes

- Create `.agents/skills/execute/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Execution flow:
  1. Read the plan phase (if one exists)
  2. Decompose into sub-tasks if large enough
  3. Work in a worktree
  4. Implement, using sub-agents for independent sub-tasks
  5. Verify: tests, lint, git diff against expected changes, plan phase completeness check
  6. Commit with conventional message
- Delegation heuristics: when to self-execute vs spawn sub-agents
- Scope discipline: only change what the phase specifies, flag anything else
- This skill is loaded alongside the adapter-configured domain skill by the dispatcher (execute session assembly, Phase 3)

## Data Structures

- Input: cook prompt (backlog item ID + rationale) + plan phase file (if exists)
- Output: committed code changes in the worktree

## Verification

- Static: SKILL.md has frontmatter, principles, execution flow, delegation heuristics, scope discipline
- Static: `noodle:` frontmatter exists
- Runtime: Spawn an execute session for a plan phase. Verify:
  - Agent reads the plan phase file
  - Agent works in a worktree
  - Agent runs tests and lint before committing
  - Agent checks all plan phase requirements are addressed
  - Commit message references the backlog item
  - Scope discipline: only expected files modified
