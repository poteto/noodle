Back to [[archived_plans/26-context-usage-tracking/overview]]

# Phase 6: Add context audit to meditate skill

## Goal

Teach meditate's Reviewer agent to consume `mise.json`'s `ContextSummary` and flag the highest-footprint skills for investigation. This extends the existing skill review with data-driven context analysis.

## Changes

- **`.agents/skills/meditate/SKILL.md`** — Update step 3 (Reviewer):
  - Reviewer should read `mise.json` if available (in addition to brain/skills snapshots)
  - In the Skill Review section, compare each skill's context footprint from `ContextSummary`
  - Flag skills with above-average compression rates or peak context usage

- **`.agents/skills/meditate/references/agents.md`** — Update Reviewer spec:
  - Add `mise.json` as optional input
  - Add context efficiency check to the skill review checklist
  - Produce recommendations: tighten description, split skill body, move content to references

Use the `skill-creator` skill when editing the meditate skill spec.

## Data structures

- No new types. Consumes `ContextSummary` from `mise.json`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Skill design and agent prompt writing |

## Verification

- Meditate SKILL.md and references/agents.md updated
- Reviewer prompt spec includes context efficiency check
- Manual: run meditate with a mise.json that has context stats → report includes context findings
