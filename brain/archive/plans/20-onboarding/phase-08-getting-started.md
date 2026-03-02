Back to [[plans/20-onboarding/overview]]

# Phase 8 — Getting-Started Tutorial

## Goal

Write a step-by-step tutorial that takes a new user from zero to a running Noodle instance. This is the critical path doc — the thing linked from the README.

## Changes

- **`docs/getting-started.md`** — Tutorial covering:
  1. Prerequisites (tmux, Claude Code or Codex CLI)
  2. Install Noodle (Homebrew)
  3. Initialize a project (`noodle start` in an existing repo)
  4. What just happened (explain the scaffolded files: `.noodle.toml`, `brain/`, `.noodle/`)
  5. Add your first backlog item (edit `brain/todos.md`)
  6. Watch Noodle work (the schedule → cook cycle)
  7. Review the output (session summaries, merged changes)
  8. Next steps (links to concept docs, examples, config reference)

## Data structures

None — tutorial prose only. Config reference is already shipped in Phase 3.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Tutorial quality is critical for first impressions |

Apply `unslop` skill to all prose.

## Verification

### Static
- Page builds without errors
- All internal links resolve (concept docs, config reference, examples)

### Runtime
- Follow the tutorial in a fresh directory with Noodle installed via Homebrew
- Every step produces the described result
- No steps require knowledge not yet introduced
