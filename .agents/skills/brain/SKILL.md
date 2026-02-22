---
name: brain
description: Update the brain (Obsidian vault at brain/). Use when writing, editing, or deleting brain files — whether during reflection, planning, or any task that persists knowledge. Triggers on brain/ file modifications, "update the brain", "add to brain", "brain note", or when routing learnings to persistent memory.
---

# Brain

Persistent memory across sessions. Obsidian vault at `brain/`.

The brain is the foundation of the entire workflow — every agent, skill, and session reads it. Low-quality or speculative content degrades everything downstream. Before adding anything, ask: "Does this genuinely improve how the system operates?" If the answer isn't a clear yes, don't write it.

## Before Writing

Read `brain/index.md` first. Then read the relevant entrypoint for your topic:

- `brain/principles.md` for principle updates
- `brain/delegation.md` for delegation/orchestration learnings
- `brain/plans/index.md` for plan-related notes

For directories without a dedicated index file yet (`brain/codebase/`, `brain/audits/`, `brain/vision/`), scan nearby files directly and edit an existing note when possible.

## Structure

```
brain/
├── index.md              ← root entry point, links to everything
├── principles.md         ← index for principles/
├── delegation.md         ← index for delegation/
├── todos.md              ← product backlog
├── vision/               ← product/system vision docs
├── principles/           ← engineering and design principles
├── delegation/           ← delegation and orchestration learnings
├── codebase/             ← project-specific knowledge and gotchas
├── plans/                ← numbered feature plans (02-arrow-bindings/, etc.)
└── audits/               ← design and quality audits (date-prefixed filenames, e.g. 2026-02-20-quality-audit.md)
```

**Rules:**

- One topic per file. `brain/codebase/noodle-spawn-gotchas.md`, not a mega-file.
- Maintain existing index entrypoints: `brain/index.md`, `brain/principles.md`, `brain/delegation.md`, `brain/plans/index.md`.
- If you introduce a new top-level category, add an index-style entrypoint for it (links only, no inlined content).
- `brain/index.md` is the root. Every brain file must be reachable from it.
- File names: lowercase, hyphenated. `worktree-gotchas.md`.
- Plan directories: prefixed with the todo ID from `brain/todos.md`. `02-arrow-bindings/` with `overview.md` and `phase-N-*.md` inside.

## Wikilinks

Format: `[[section/file-name]]`. Resolution order: same directory, then relative path, then vault root. Heading anchors (`[[file#heading]]`) are stripped during resolution.

## Writing Style

- Bullets over prose. No preamble.
- Plain markdown with `# Title`. No Obsidian frontmatter.
- Keep notes under ~50 lines. Split if longer.

## After Writing

Update `brain/index.md` for any files you added or removed. Also update the relevant entrypoint (`brain/principles.md`, `brain/delegation.md`, or `brain/plans/index.md`) when applicable. Keep indexes link-only and scannable.

## Durability Test

Ask: "Would I include this in a prompt for a *different* task?"

- **Yes** → write to `brain/`. It's durable knowledge.
- **No, it's plan-specific** → update the plan's phase doc instead.
- **No, it's a skill issue** → update the skill file directly.
- **No, it needs follow-up work** → file a todo with the `todo` skill.

## Maintenance

- Delete outdated or subsumed notes.
- Delete random-named files in `brain/plans/` — stale plan-mode artifacts.
- Merge overlapping notes before adding new ones.
