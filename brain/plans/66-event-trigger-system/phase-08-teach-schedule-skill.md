Back to [[plans/66-event-trigger-system/overview]]

# Phase 8 — Teach Schedule Skill to React to Events

## Goal

Update the schedule skill prompt to read `recent_events` from the mise brief and decide how to react. No Go code changes — this is purely a skill prompt update.

## Changes

- **`.agents/skills/schedule/SKILL.md`** — add a section explaining `recent_events`:
  - What internal events mean (stage completed/failed, merge, quality verdict, etc.)
  - What external events look like (arbitrary types from `noodle event emit` — e.g., `ci.failed`, `deploy.completed`)
  - How to interpret them for scheduling decisions
  - Example reactions: after `stage.failed` → consider creating an oops/debugging order; after `order.completed` → consider follow-up work; after `worktree.merged` → consider post-merge tasks; after `ci.failed` → consider a fix or investigation order
  - Emphasize: events are context, not commands. The agent decides what matters. External event types are user-defined and may be unfamiliar — interpret them from context.

Use the `skill-creator` skill when writing the updated schedule skill.

## Data Structures

None — skill prompt only.

## Routing

Provider: `claude`, Model: `claude-opus-4-6` (skill writing requires judgment)

## Verification

- Read the updated skill file and verify it references `recent_events`
- Manual test: run a full cycle, verify the schedule agent's output shows awareness of events in the mise brief
- The skill prompt should not prescribe mechanical reactions — it should frame events as context for judgment
- Verify the prompt mentions both internal lifecycle events and external user-defined events
