Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 11: Verification and Smoke Tests

## Goal

Run the full test suite, update the Playwright e2e smoke tests for the new channel layout, add new smoke test cases, and verify everything works end-to-end. This is the final gate before the redesign ships.

## Skills

Invoke `react-best-practices`, `ts-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Requires judgment on what to test and how to verify

## Changes

### Modify
- `e2e/ui/smoke.spec.ts` — rewrite existing tests for channel layout:
  - **Replace** "board loads with columns" → "channel layout loads with sidebar, feed, and context panel"
  - **Update** assertions: sidebar renders NOODLE header, nav links (Dashboard, Live Feed, Tree, Reviews), scheduler channel
  - Keep snapshot API and config API tests (unchanged)
  - Keep SSE endpoint test (unchanged)

### Add new smoke test cases
- `e2e/ui/smoke.spec.ts` — add:
  - "sidebar shows active agents from snapshot" — verify agent channels appear in sidebar
  - "channel switching updates feed panel" — click agent channel, verify feed header changes
  - "tree route loads" — navigate to `/tree`, verify SVG container renders
  - "dashboard route loads" — navigate to `/dashboard`, verify stats bar and session grid render
  - "steer input sends control command" — type in input, submit, verify POST to /api/control
  - "review banner shows for pending reviews" — if snapshot has pending reviews, verify merge/reject buttons visible

### Run full verification
1. `cd ui && pnpm tsc --noEmit` — TypeScript compilation
2. `cd ui && pnpm test` — all Vitest unit tests pass
3. `go test ./...` — all Go tests pass
4. `go vet ./...` — no issues
5. `sh scripts/lint-arch.sh` — architecture lint
6. `pnpm build` — production build succeeds
7. `pnpm test:smoke` — e2e smoke tests pass against running instance

## Verification

### Static
- All unit tests pass (`pnpm test` exits 0)
- All Go tests pass (`go test ./...` exits 0)
- TypeScript compiles cleanly
- Production build succeeds

### Runtime
- All Playwright smoke tests pass
- Manual walkthrough: load app → sidebar shows channels → click agent → see conversation → switch to tree → see graph → switch to dashboard → see history → review flow works
- No console errors in browser
- SSE reconnection works (stop/start Go server, UI reconnects)
