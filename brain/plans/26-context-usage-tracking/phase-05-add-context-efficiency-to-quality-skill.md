Back to [[plans/26-context-usage-tracking/overview]]

# Phase 5: Add context efficiency to quality skill

## Goal

Teach the quality gate to flag sessions that hit compression or used disproportionate context. This creates the feedback loop: wasteful sessions get flagged → prioritize deprioritizes or the agent investigates.

## Changes

- **`.agents/skills/quality/SKILL.md`** — Add a "Context Efficiency" section to the review checklist:
  - Flag if session hit compression (`CompressionCount > 0`)
  - Flag if peak context exceeded 80% of budget
  - Note the skill name and suggest investigation if context usage is abnormally high relative to the task complexity
  - Include context metrics in the verdict output

- **`loop/quality.go`** — Pass context metrics from `meta.json` to the quality skill's prompt (already passes session info — extend with new fields)

Use the `skill-creator` skill when editing the quality skill spec.

## Data structures

- Quality verdict schema (`.noodle/quality/<id>.json`) may gain an optional `context_flags []string` field for structured context feedback

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Skill design requires judgment about thresholds and wording |

## Verification

- Quality skill SKILL.md updated with context efficiency section
- `go test ./loop/...` — existing tests pass
- New test: quality spawn includes context metrics in prompt when available
- Manual: run quality on a session with compression → verdict mentions context flag
