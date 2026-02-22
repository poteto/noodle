Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 2: Delete Parent-Project-Specific Files

## Goal
Strip out everything that belongs to the parent project (the product), leaving only what's relevant to Noodle development.

## Changes
- Delete the parent project's frontend: `src/`, `src-tauri/`, `public/`, `index.html`, Vite/React/TypeScript config files (`vite.config.ts`, `tsconfig*.json`, `tailwind.config.*`, `postcss.config.*`, etc.)
- Delete parent project package files: `package.json`, `pnpm-lock.yaml`, `node_modules/`
- Delete parent project CI workflows that are frontend/Tauri-specific (keep any that are relevant to Go/Noodle)
- Delete parent-project-specific brain plans (the product plans like arrow-bindings, canvas-polish, etc.) — keep plan 1 and any tooling plans
- Delete parent-project-specific todos from `brain/todos.md` (product direction, core experience, canvas polish, etc.) — keep the Tooling section
- Delete parent-project-specific skills that are purely frontend/product (e.g., `frontend-design`, `interaction-design`, `prototype`, `react-best-practices`, `web-design-guidelines`, `profiling`) — keep skills that are useful for Noodle development (e.g., `debugging`, `commit`, `worktree`, `skill-creator`, `plan`, `review`, `reflect`, `brain`, `todo`, `noodle`, `bubbletea-tui`)
- Keep `.claude/settings.json`, hooks, and general-purpose agent definitions
- Keep `brain/principles/` — these apply to Noodle development too

## Data Structures
- No new types.

## Verification

### Static
- No `src/`, `src-tauri/`, `public/`, or frontend config files remain
- No `package.json` or `node_modules/`
- Skills directory contains only Noodle-relevant skills
- `brain/todos.md` contains only Noodle-relevant items
- `brain/plans/` contains only plan 1

### Runtime
- The parent project is completely untouched
