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

### Prioritize reasons

The prioritize agent receives a `reason` in its input that explains _why_ it was invoked. Different reasons produce different behavior:

| Reason | When | Blocking | Agent focus |
|--------|------|----------|-------------|
| `startup` | Fresh boot, no queue exists | Yes — nothing can run until the first queue is built | Full survey — read all backlog items, all plan states, build queue from scratch |
| `backlog_changed` | Todo added, edited, or completed via adapter | No — existing queue items can continue executing in parallel | Incremental — slot the change into the existing queue, don't rebuild everything |
| `plan_created` | A plan skill produced new phases | No — existing queue items can continue executing in parallel | Schedule new phases relative to existing queue items |
| `quality_rejected` | Quality rejected a cook after max retries | No — existing queue items can continue | Rescope the item, deprioritize, or surface to chef for review |

The reason and any associated context (e.g. which todo changed, which plan was created) are included in the mise brief so the agent can focus its judgment appropriately.

### Push-based backlog notification (Go infrastructure)

When the backlog adapter reports mutations (new items, edits, completions), the loop should automatically insert a prioritize task at the top of the queue with reason `backlog_changed`. This is a Go code change in the loop — not in the skill itself. The loop detects backlog changes by diffing the adapter sync output against the previous state.

## Data Structures

- Input: `.noodle/mise.json` — structured brief with backlog state, resource state, active sessions, recent history, **reason** (startup | backlog_changed | plan_created | quality_rejected), and reason context
- Output: `.noodle/queue.json` — `{ generated_at, items: [{ id, task_key, title, provider, model, skill, review, rationale }] }`

## Verification

- Static: SKILL.md has frontmatter, principles section, contract section, process with rationale requirements
- Runtime: Spawn a test prioritize session with a sample mise.json containing mixed priorities, a pending chef review, and a repeatedly-failed item. Verify:
  - Chef review items come first when pending
  - Foundation items precede feature items
  - Rationale cites specific ordering principles
  - Failed items are flagged for rescoping, not blindly retried
  - Model routing uses cheapest viable option
  - Reason is included in the input and influences agent behavior (startup = full survey, backlog_changed = incremental)
  - Startup prioritize is blocking (loop waits); backlog_changed and plan_created are non-blocking (cooks continue in parallel)
