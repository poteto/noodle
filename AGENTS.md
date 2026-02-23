# Noodle

Open-source AI coding framework. Skills as the only extension point, LLM-powered prioritization, kitchen brigade model. Built with Go.

## Brain

The `brain/` directory is an Obsidian vault. Use it as persistent memory:

- **Read relevant brain files before you act.** The SessionStart hook injects the brain index — read the files that apply to your task, don't just glance at filenames.
- **When to write:** After making a mistake, getting corrected, or learning something notable about the codebase.
- **Structure:** One topic per file. Group in directories with index files that link via `[[wikilinks]]` — no inlined content in indexes.
- **Maintain:** Delete outdated notes and random-named files in `brain/plans/` (stale plan-mode artifacts).

## Task Execution

- **Respect user ordering:** When the user provides a bullet or numbered list, create Tasks and follow the specified order. You may suggest reordering if it would be more efficient or logical, but don't reorder silently.

## Concurrent Sessions & Commits

- Multiple sessions run simultaneously. Default to a linked git worktree created via Skill(worktree) (usually `.worktrees/<name>`), not the primary checkout at repo root. Use Skill(worktree) and Skill(commit) for significant or autonomous work.

## Coding Conventions

- **Error messages**: describe failure state ("session not found"), not expectations ("session must exist").
- **Never use haiku model** — use sonnet when a faster/cheaper model is needed.

## Platform Compatibility

This project targets cross-platform (macOS local dev, Windows/Linux CI). Avoid bash 4+ features like `declare -A` (macOS ships bash 3.2). Prefer POSIX-compatible shell or use Go for non-trivial scripts.
