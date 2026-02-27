Back to [[archive/plans/73-testing-strategy/overview]]

# Phase 3: UI Component Tests

## Goal

Test the critical component logic paths that agents are most likely to break when modifying the UI. Focus on data-driven rendering, user interaction handlers, and optimistic state — not visual styling.

## Changes

**Test files to create (one per component group):**

**`ui/src/components/Board.test.tsx`** — Test optimistic reducer: `applyOptimisticSnapshot` for `stop`, `merge`, `reject`, `requeue`, `reorder`, `move-to-cooking` (including capacity gate that blocks when at max cooks), and the `request-changes` no-op case. Test keyboard shortcut registration (`n`, `p`) and suppression when focus is inside an input or select.

**`ui/src/components/OrderCard.test.tsx`** — Test rendering with various order shapes: single-stage, multi-stage, with on_failure stages. Test `currentStage` derivation: picks first `active` or `merging` stage, falls back to first `pending`. Test stage rail renders inside the card.

**`ui/src/components/AgentCard.test.tsx`** — Test rendering with session data: cost display, duration, context window usage bar. Test `CurrentAction` extraction from event types. Test stop button fires correct control callback.

**`ui/src/components/StageRail.test.tsx`** — Test compression logic: ≤8 stages renders all dots, >8 stages compresses middle with count indicator. Test status dot coloring for each stage status. Test that clicking a stage dot with a `session_id` triggers chat panel selection.

**`ui/src/components/ReviewActions.test.tsx`** — Test merge, reject, request-changes button callbacks send correct control commands.

**`ui/src/components/LoopControls.test.tsx`** — Test pause/resume toggle sends correct control action based on current loop state.

**`ui/src/components/TaskEditor.test.tsx`** — Test form submission builds correct enqueue control command.

**`ui/src/client/api.test.ts`** — Test `normalizeSnapshot`: null arrays → empty arrays. Mock `fetch` to test `fetchSnapshot`, `fetchConfig`, `fetchReviewDiff`, and `sendControl`. Test error handling on non-OK responses.

**`ui/src/client/sse.test.ts`** — Test SSE connection lifecycle: open → message → status transitions. Test reconnect on error (2s delay). Test malformed message handling.

**Shared test factory:**

**`ui/src/test-utils.ts` (new)** — Snapshot/session/order factory functions for building test data. E.g., `buildSnapshot({ active: [buildSession({ id: "s1" })] })`. All fields have sensible defaults, overrides are partial.

## Data Structures

- `buildSnapshot()`, `buildSession()`, `buildOrder()`, `buildStage()` — factory functions returning typed test data with sensible defaults

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical test writing against clear component contracts |

## Verification

### Static
- `pnpm --filter ui tsc --noEmit`
- `pnpm --filter ui test` — all tests pass

### Runtime
- Each test file covers the primary render path and at least one interaction
- Test factories produce valid typed data (TS compiler enforces this)
- Verify tests catch regressions: temporarily break a component, confirm test fails
