Back to [[plans/20-onboarding/overview]]

# Phase 2 — Lean README Rewrite

## Goal

Audit existing root-level docs, then rewrite README.md as a focused quick-start funnel. Subtract first — move or delete redundant files before writing new content.

## Changes

### Subtraction (first)

Audit and decide the fate of every existing docs file:

- **`INSTALL.md`** — Currently targets agents, not humans. Merge any useful content (prerequisites, project setup) into the getting-started tutorial scope. **Delete.**
- **`PHILOSOPHY.md`** — Good content about the "why." Move to docs site as `docs/concepts/philosophy.md` or fold into the brain/modes concept pages. **Delete from root.**
- **`AGENTS.md`** — Duplicate of CLAUDE.md. **Delete.**

### Addition

- **`README.md`** — Rewrite. Structure: one-liner description, install (Homebrew), first run (`noodle start`), what happens next (link to getting-started), link to docs site, contributing basics

## Data structures

None — markdown only.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Writing quality matters for the first thing users see |

Apply `unslop` skill to all prose.

## Verification

### Static
- README renders correctly on GitHub (check markdown preview)
- All links resolve (docs site links, install links)
- No orphaned root-level docs files

### Runtime
- Follow the README steps in a fresh directory with Noodle installed via Homebrew
- `noodle start` in a fresh project scaffolds the expected structure
- No root-level docs files remain that duplicate docs site content
