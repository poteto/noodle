Back to [[archived_plans/47-delete-go-tui/overview]]

# Phase 4 — Delete bubbletea-tui skill

## Goal

Remove the Bubble Tea TUI skill — no TUI means no need for TUI development guidance.

## Changes

- Delete `.claude/skills/bubbletea-tui/` directory entirely:
  - `SKILL.md`
  - `references/elements.md`
  - `references/production-rules.md`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

### Static
- Directory deleted
- No remaining references to `bubbletea-tui` in `.claude/` config files
