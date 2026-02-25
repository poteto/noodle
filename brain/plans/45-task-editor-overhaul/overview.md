---
id: 45
created: 2026-02-24
status: ready
---

# Task Editor Overhaul

## Context

The task editor ("New Task" modal) has several UX gaps: redundant fields (Type and Skill overlap), hardcoded enum values cycled with left/right arrows instead of proper dropdowns, single-line prompt input with no visible cursor, and no up/down field navigation. This plan redesigns the task editor from first principles — if we were building this form fresh with these requirements, what would we build?

## Scope

**In scope:**
- Merge Type + Skill into a single "Skill" field backed by the task type registry (dynamic, not hardcoded)
- Show middle-truncated skill description alongside the skill name
- Overlay dropdown for all enum fields (Skill, Model, Provider)
- Multiline prompt textarea with visible text cursor
- Context-sensitive up/down navigation (move between fields on enum fields, move cursor within prompt textarea)
- Updated footer hints

**Out of scope:**
- Changing the control command protocol or queue item schema (TaskKey/Skill still sent as before)
- Skill search/filtering within dropdowns (can be added later)
- Provider-dependent model filtering (e.g. showing only Claude models when Claude provider is selected)

## Constraints

- Bubble Tea v2 (`charm.land/bubbletea/v2`) and Lipgloss v2 — already in use
- Components are dumb: expose methods, main model owns routing (per bubbletea-tui skill)
- ANSI-safe string operations via `github.com/charmbracelet/x/ansi` for truncation
- The `TaskEditor` struct is embedded in the main `Model` — keep it simple, no separate tea.Model
- Dynamic skill list needs plumbing from `loop.Loop` → `tui.Model` → `TaskEditor`

### Alternatives considered

**Dropdown approach:**
1. **Inline expansion** — field expands in place, pushes other fields down. Simpler rendering but awkward layout shifts.
2. **Overlay popup** — floating list appears over other fields. Better UX, standard dropdown behavior. **Chosen.**
3. **Full-screen picker** — overlay takes whole modal. Overkill for 3-5 items.

**Prompt input approach:**
1. **bubbles/v2 textarea** — mature component with cursor, scrolling, word wrap. But heavyweight and forces tea.Model interface on the editor.
2. **Custom textarea** — extend existing char-by-char handling with cursor position tracking and multiline support. Lighter, fits existing pattern. **Chosen** — the editor is a struct with methods, not a tea.Model, so using bubbles textarea would require rearchitecting.

## Applicable skills

- `bubbletea-tui` — component patterns, styling, ANSI-safe operations, focus states

## Phases

1. [[plans/45-task-editor-overhaul/phase-01-restructure-key-routing]]
2. [[plans/45-task-editor-overhaul/phase-02-consolidate-fields-and-dynamic-skills]]
3. [[plans/45-task-editor-overhaul/phase-03-dropdown-overlay-component]]
4. [[plans/45-task-editor-overhaul/phase-04-wire-dropdowns-to-enum-fields]]
5. [[plans/45-task-editor-overhaul/phase-05-multiline-prompt-with-cursor]]
6. [[plans/45-task-editor-overhaul/phase-06-context-sensitive-up-down-navigation]]

## Verification

```
go test ./tui/... && go vet ./...
sh scripts/lint-arch.sh
```

Visual verification: launch the TUI, press `n` to open the task editor, exercise all fields and navigation.
