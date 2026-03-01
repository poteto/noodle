Back to [[plans/20-onboarding/overview]]

# Phase 6 — Concept Doc: Brain & Modes

## Goal

Write concept pages for the brain vault and operating modes. These are the two remaining core concepts users need to understand.

## Changes

- **`docs/concepts/brain.md`** — New page covering:
  - What the brain is (Obsidian-compatible vault, persistent memory)
  - Structure: principles, codebase notes, plans, todos
  - How agents interact with it (reflect, meditate, plan skills)
  - The self-learning loop: work → reflect → encode → improve
  - Wikilinks and cross-referencing

- **`docs/concepts/modes.md`** — New page covering:
  - The three modes: auto, supervised, manual
  - What each mode controls (merge behavior, scheduling autonomy)
  - When to use which mode
  - The kitchen brigade metaphor (chef, schedule, cook, quality)
  - How the human fits in as supervisor, not operator

## Data structures

None — conceptual documentation only.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Conceptual writing, needs to convey philosophy clearly |

Apply `unslop` skill. Draw from `PHILOSOPHY.md` and `brain/vision.md` for source material.

## Verification

### Static
- Pages build without errors
- Documented mode names exactly match the set validated in `config/parse.go` (currently: `auto`, `supervised`, `manual`)
- Brain directory structure described matches `startup/firstrun.go` scaffold output

### Runtime
- Brain structure described matches what `noodle start` creates on first run
- Mode behavior descriptions match observable behavior (e.g. supervised mode prompts before merge)
