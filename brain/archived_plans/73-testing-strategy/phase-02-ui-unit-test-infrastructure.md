Back to [[archived_plans/73-testing-strategy/overview]]

# Phase 2: UI Unit Test Infrastructure

Depends on phase 1 (imports generated types).

## Goal

Set up Vitest + React Testing Library so UI components and utility functions can be tested. Write initial tests for the pure data layer to prove the setup works.

## Changes

**`ui/package.json`** — Add dev dependencies: `vitest`, `@testing-library/react`, `@testing-library/jest-dom` (v6+), `@testing-library/user-event`, `jsdom`. Add scripts: `"test": "vitest run"`, `"test:watch": "vitest"`. Note: `"test"` must use `vitest run` (single-pass), not `vitest` (watch mode), to avoid hanging in CI.

**`ui/vitest.config.ts` (new)** — Extend the existing Vite config rather than re-specifying path aliases (the project already uses `vite-tsconfig-paths` which handles `~/*` → `./src/*`). Add jsdom environment, setup file reference. Note: strip the `TanStackRouterVite` plugin when extending — it scans `src/routes/` to regenerate `routeTree.gen.ts` and will cause noisy output and potential flakiness in test runs.

**`ui/src/test-setup.ts` (new)** — Import `@testing-library/jest-dom/vitest` for DOM matchers (v6 import path).

**`ui/src/client/format.test.ts` (new)** — Tests for `formatCost`, `formatDuration`, `middleTruncate` (these live in `ui/src/client/format.ts`). Pure functions — easiest first test, validates setup works.

**`ui/src/client/types.test.ts` (new)** — Tests for `deriveKanbanColumns`. Edge cases to cover:
- Empty snapshot (all arrays empty)
- `active_order_ids` contains IDs not present in `orders` (silently ignored)
- Orders split correctly between queued and active
- `pending_reviews` flows to review column

## Data Structures

- No new types. Tests consume existing `Snapshot`, `Order`, `Session` types from generated + hand-written files.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Boilerplate config + straightforward tests |

## Verification

### Static
- `pnpm --filter ui tsc --noEmit` — TS compiles
- `pnpm --filter ui test` — all tests pass

### Runtime
- `pnpm --filter ui test:watch` — watch mode works for iterating
- Add a failing test, confirm it reports failure correctly
