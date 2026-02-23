Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 1: Prioritize — Rewrite Queue Scheduler

## Goal

Rewrite `.agents/skills/prioritize/SKILL.md` to incorporate CEO scheduling judgment and ground it in engineering principles. This is the sous chef's brain — the most important skill in the cook loop.

## Current State

- `.agents/skills/prioritize/SKILL.md` — functional queue builder but lacks scheduling judgment framework, rationale depth, and principle grounding
- `skills/sous-chef/SKILL.md` — old name, lean stub (renamed in Phase 13)

## Patterns to Incorporate

From **CEO**:
- **Foundation-before-feature ordering** — infrastructure, shared types, and scaffolding before features. Backlog order is dependency-aware.
- **Cheapest execution mode** — schedule the cheapest provider/model that can finish the job safely. Don't default to expensive models for mechanical work.
- **Explicit rationale** — every queue item gets a rationale explaining which ordering rule drove its placement. Rationale is auditable.
- **Fresh context** — each prioritize cycle starts clean. No accumulated drift from previous cycles.
- **Work around blockers** — when approvals or reviews are pending, schedule unblocked work instead of idling. Never wait when alternatives exist.
- **Timebox high-risk work** — rescope instead of infinite retries.

## Principles

- [[principles/cost-aware-delegation]] — cheapest mode that finishes safely
- [[principles/foundational-thinking]] — infrastructure before features
- [[principles/subtract-before-you-add]] — cut low-value items before adding new work
- [[principles/never-block-on-the-human]] — schedule around blocked approvals
- [[principles/guard-the-context-window]] — keep the skill lean; mise.json is the only input

## Changes

- Rewrite `.agents/skills/prioritize/SKILL.md` — **use the `skill-creator` skill**
- Add Principles section with `[[wikilinks]]` to relevant principles
- Add scheduling judgment framework from CEO (not just mechanical ordering rules)
- Strengthen rationale requirements — rationale must cite the ordering principle that drove placement
- Add cost-awareness to model routing — explain when to use expensive vs cheap models
- Add blocker-avoidance logic — when chef review is pending, schedule unblocked work
- Add timebox guidance — items that have failed repeatedly should be rescoped, not retried
- This skill is autonomous-only — no AskUserQuestion, no interactive sections

### Situational awareness

The prioritize agent doesn't need an explicit "reason" passed by Go infrastructure. Instead, it infers context from the mise brief data:

- **Empty queue + no active sessions** → startup: full survey, build from scratch
- **Quality rejection in recent history** → rescope the item, deprioritize, or surface to chef
- **New backlog items not in queue** → slot them into the existing queue
- **New plan phases not yet scheduled** → schedule phases relative to existing items

The mise brief already contains backlog state, active sessions, recent history, and plan data. The skill reads these files and makes its own judgment — no Go-level change detection or reason routing needed. This keeps the Go core as a thin data assembler and lets the skill handle all scheduling intelligence.

## Data Structures

- Input: `.noodle/mise.json` — structured brief with backlog state, plan state, resource state, active sessions, recent history
- Output: `.noodle/queue.json` — `{ generated_at, items: [{ id, task_key, title, provider, model, skill, review, rationale }] }`

## Verification

- Static: SKILL.md has frontmatter, principles section, contract section, process with rationale requirements
- Runtime: Spawn a test prioritize session with a sample mise.json containing mixed priorities, a pending chef review, and a repeatedly-failed item. Verify:
  - Chef review items come first when pending
  - Foundation items precede feature items
  - Rationale cites specific ordering principles
  - Failed items are flagged for rescoping, not blindly retried
  - Model routing uses cheapest viable option
  - Agent infers context from brief state (empty queue → full survey, quality rejections → rescoping)
