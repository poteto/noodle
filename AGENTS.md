# Noodle

Open-source AI coding framework. Skills as the only extension point, LLM-powered prioritization, kitchen brigade model. Built with Go.

## Brain

The `brain/` directory is an Obsidian vault. Use it as persistent memory:

- **Read relevant brain files before you act.** The SessionStart hook injects the brain index — read the files that apply to your task, don't just glance at filenames.
- **When to write:** After making a mistake, getting corrected, or learning something notable about the codebase.
- **Structure:** One topic per file. Group in directories with index files that link via `[[wikilinks]]` — no inlined content in indexes.
- **Maintain:** Delete outdated notes and random-named files in `brain/plans/` (stale plan-mode artifacts).

## Code Design

**CRITICAL:** For each proposed change, examine the existing system and redesign it into the most elegant solution that would have emerged if the change had been a foundational assumption from the start.

## Task Execution

- **Parallelize by default:** When using Tasks, always evaluate whether tasks can run in parallel. Spawn an agent team if so.
- **Respect user ordering:** When the user provides a bullet or numbered list, create Tasks and follow the specified order. You may suggest reordering if it would be more efficient or logical, but don't reorder silently.

## Concurrent Sessions & Commits

**CRITICAL:** Multiple sessions run simultaneously. Default to a linked git worktree created via Skill(worktree) (usually `.worktrees/<name>`), not the primary checkout at repo root. Primary checkout is allowed only for interactive single-agent small changes; use Skill(worktree) and Skill(commit) for significant or autonomous work.

## Coding Conventions

- **Go project** — all production code is Go. Use `go test`, `go build`, `go vet`.
- **Error messages**: describe failure state ("session not found"), not expectations ("session must exist").
- **Never use haiku model** — use sonnet when a faster/cheaper model is needed.
- **Prefer workflows over skills for CI** — if automation can run headlessly, make it a GitHub Actions workflow.

## Cook Sessions

When running non-interactively (`claude -p` with `COOK=1`):
- Never use `AskUserQuestion` — all scope and constraints are in the initial prompt.
- Never wait for human confirmation — proceed autonomously.
- Route findings and decisions to `brain/` files or `brain/todos.md`.
- Conclude by writing outputs, not by presenting to a user.
- **PermissionRequest timeout in `-p` mode is non-blocking/allow.** Cook runs do not stall on timeout.

## Platform Compatibility

This project targets cross-platform (macOS local dev, Windows/Linux CI). Avoid bash 4+ features like `declare -A` (macOS ships bash 3.2). Prefer POSIX-compatible shell or use Go for non-trivial scripts.

## Repository Structure

- `old_noodle/` — reference Go codebase from the original Noodle implementation (will be rewritten)
- `brain/` — Obsidian vault (persistent memory, plans, principles)
- `.agents/skills/` — Claude Code skills (the only extension point)
- `.noodle/config.toml` — project-level Noodle configuration
- `scripts/` — development scripts (lint, etc.)
