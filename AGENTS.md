# Noodle

Open-source AI coding framework. Skills as the only extension point, LLM-powered scheduling, kitchen brigade model. Built with Go.

## Brain

The `brain/` directory is an Obsidian vault — persistent memory across sessions.

- **Read first.** Read brain files relevant to your task before acting.
- **Write** after mistakes, corrections, or notable codebase learnings.
- **Structure:** One topic per file. Directories with `[[wikilink]]` indexes — no inlined content.
- **Maintain:** Delete outdated notes and stale plan-mode artifacts in `brain/plans/`.

## Workflow

- **User ordering:** Follow bullet/numbered list order. Don't reorder silently.
- **Isolation:** Multiple sessions run concurrently — use `Skill(worktree)` and `Skill(commit)`.

## Coding

- **Error messages:** Describe failure state ("session not found"), not expectations ("session must exist").
- **Cross-platform** (macOS/Windows/Linux): No bash 4+ features. Prefer POSIX shell or Go.
- **No backward compatibility** by default. No `omitempty` shims, no legacy fallbacks, no dual-path support. Only add compat when explicitly requested.
