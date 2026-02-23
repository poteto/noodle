Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 4: Prioritize — Queue Scheduler

## Goal

Rewrite `.agents/skills/prioritize/SKILL.md` to incorporate CEO scheduling judgment, plans-as-precondition, and schedule reading. This is the sous chef's brain — the most important skill in the cook loop.

## Current State

- `.agents/skills/prioritize/SKILL.md` — functional queue builder but lacks scheduling judgment, rationale depth, and principle grounding

## Patterns to Incorporate

From **CEO**:
- Foundation-before-feature ordering
- Cheapest execution mode that finishes safely
- Explicit rationale for every scheduling decision
- Fresh context each cycle — no accumulated drift
- Work around blockers, don't idle
- Timebox high-risk work — rescope instead of infinite retries

## Principles

- [[principles/cost-aware-delegation]] — cheapest mode that finishes safely
- [[principles/foundational-thinking]] — infrastructure before features
- [[principles/subtract-before-you-add]] — cut low-value items before adding new work
- [[principles/never-block-on-the-human]] — schedule around blocked approvals
- [[principles/guard-the-context-window]] — keep the skill lean; mise.json is the only input

## Changes

- Rewrite `.agents/skills/prioritize/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = true`
- Include `references/mise-schema.md` and `references/queue-schema.md`
- Add scheduling judgment framework from CEO patterns
- Rationale must cite the ordering principle that drove placement
- Cost-aware model routing guidance
- Blocker-avoidance: schedule unblocked work when approvals are pending
- Timebox: items that have failed repeatedly should be rescoped

### Plans as precondition

Only schedule execution for backlog items with a linked plan. Unplanned items are excluded from the queue — they require user action. Note which items were skipped so the TUI can surface "action needed."

### Schedule_hint reading

Read the `schedule` from each discovered task type in the mise brief. Use these hints alongside session history and backlog state to decide when to schedule each task type. This is how user-defined task types get scheduled — the prioritize skill reads their hints and exercises judgment.

### Situational awareness

The agent infers context from the mise brief — no explicit reason from Go:

- **Empty queue + no active sessions** → full survey, build from scratch
- **Quality rejection in history** → rescope, deprioritize, or surface to chef
- **New backlog items with plans** → slot into existing queue
- **New plan phases** → schedule relative to existing items
- **Unplanned items** → skip, note as needing user action

## Data Structures

- Input: `.noodle/mise.json` — backlog, plans, task types, sessions, history, quality verdicts
- Output: `.noodle/queue.json` — `{ generated_at, items: [{ id, task_key, title, provider, model, skill, review, rationale }] }`

## Verification

- Static: SKILL.md has frontmatter, principles, contract, process with rationale requirements
- Static: `noodle:` frontmatter exists with `blocking: true`
- Static: `references/` includes mise and queue schemas
- Runtime: Spawn a prioritize session with mixed priorities, quality rejections, and unplanned items. Verify:
  - Foundation items precede feature items
  - Rationale cites specific principles
  - Failed items are rescoped, not retried
  - Model routing uses cheapest viable option
  - Unplanned items are excluded (plans-as-precondition)
  - Agent infers context from brief state
