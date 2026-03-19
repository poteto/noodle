# Verification

Every change must pass before committing. Fix and re-verify on failure. Never commit failing code.

## Full Test Suite

After each change, run the complete suite:
- `pnpm build` — compiles UI and Go binary; catches type errors in both
- `go test ./...` (or scoped to changed packages)
- `go vet ./...`
- `pnpm --filter noodle-ui test` — UI unit tests (vitest)
- `sh scripts/lint-arch.sh` — if present

Or equivalently: `pnpm check` (runs build, Go tests, vet, arch lint, and fixture tests).

## E2E Smoke Test

After integrating changes, before merging to main:
- `pnpm test:smoke` — Go e2e tests with `-tags e2e`
- In a worktree: `noodle worktree exec <name> pnpm test:smoke`
- When UI changes: write NEW test cases covering the changed interface

## Fixture Tests

When changes affect loop behavior or runtime state:
- `pnpm fixtures:loop` / `pnpm fixtures:hash`
- Update fixtures: `pnpm fixtures:loop:record` then `pnpm fixtures:hash:sync`

## Visual Verification

When changes affect UI:
- Use the Chrome tool to open the UI in browser, click through affected flows

## Scope Check

- `git diff --stat` — matches expected scope
- All checklist items addressed (plan phase, todo criteria, or ad-hoc requirements)
