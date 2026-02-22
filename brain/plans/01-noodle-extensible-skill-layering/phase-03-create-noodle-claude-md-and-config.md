Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 3: Adapt CLAUDE.md and Project Config for Noodle

## Goal
Rewrite CLAUDE.md and project config to be Noodle-specific rather than parent-project-specific.

## Changes
- Rewrite `CLAUDE.md` — replace parent-project-specific content with Noodle context: Go project conventions, the noodle CLI, brain usage, skill conventions, the kitchen brigade model. Remove references to pnpm/Vite/Tauri/React.
- Update `.claude/settings.json` — remove hooks that reference parent-project-specific paths (e.g., `enforce-pnpm.sh`). Keep brain injection, brain auto-index, and worktree hooks. Update any hook command paths that reference `$CLAUDE_PROJECT_DIR/noodle` to reference the repo root directly.
- Update `brain/index.md` and `brain/vision/noodle.md` to reflect the new repo structure and open-source vision.
- Clean up `brain/codebase/` notes — remove parent-project-specific notes (vite-gotchas, styling-architecture, tauri-plugin-mcp, etc.), keep Noodle-relevant notes.

## Data Structures
- No new Go types.

## Verification

### Static
- `CLAUDE.md` contains no references to the parent project, pnpm, Vite, Tauri, or React
- `.claude/settings.json` hooks all reference valid paths within the new repo
- Brain files reference valid wikilinks (no broken links to deleted files)

### Runtime
- Start a Claude Code session in `~/code/noodle/` — verify brain injection fires and CLAUDE.md loads correctly
