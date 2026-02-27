---
id: 73
created: 2026-02-26
status: done
---

# Testing Strategy

## Context

Noodle has decent Go unit tests and a sophisticated fixture framework, but significant gaps remain:

- **UI has zero test coverage.** 29+ React components, SSE data flow, optimistic updates, drag-and-drop — all untested. Agents modifying the UI have no way to verify their changes work.
- **Go↔TS types are manually mirrored.** `internal/snapshot/types.go` and `ui/src/client/types.ts` are hand-synced. Drift causes runtime crashes (see fix e8b4e66 for nil slices vs empty arrays). This is the class of bug that testing alone can't prevent — it needs structural elimination.
- **Fixture coverage has gaps.** The fixture framework is well-designed but only covers a subset of loop states. The snapshot builder has no fixtures at all.
- **No E2E tests spawn real agents.** The sandbox script creates test projects but doesn't verify Noodle actually works end-to-end. No test proves the full cycle: mise → schedule → dispatch → cook → complete.

## Scope

**In scope:**
- Typesafe Go↔TS API boundary (subsumes todo 71)
- UI unit test infrastructure (Vitest + Testing Library)
- UI component tests for critical logic
- Expanding Go fixture tests for uncovered state transitions
- E2E smoke test that spawns a real Codex agent

**Out of scope:**
- Testing skill updates (separate effort)
- CI integration for E2E tests (manual-only for now due to API costs)
- Browser E2E tests with mock server (drift-prone, adds little over smoke test)
- Visual regression/screenshot testing
- Performance/load testing

## Constraints

- E2E agent tests use Codex for cost efficiency — manual invocation only, not CI
- UI tests must work with Vite 7 + React 19 + TanStack Router
- Fixture framework already exists (`internal/testutil/fixturedir/`) — extend, don't replace
- Cross-platform: tests must work on macOS/Linux (Windows best-effort)

## Alternatives Considered

**Typesafe API boundary:**
1. **TypeSpec / OpenAPI codegen** — generate Go server + TS client from a shared spec. Heavy tooling, forces framework opinions on the Go HTTP layer.
2. **Go struct → JSON Schema → TS types** — generate JSON Schema from Go structs, then TS interfaces from schema. Two-step pipeline, schemas are lowest-common-denominator.
3. **Go struct → TS types directly** — generate TypeScript interfaces directly from Go structs with a Go tool like `tygo`. Single step, minimal tooling, fits the project's "subtract before you add" principle.

**Chosen: option 3 (tygo).** One `go generate` command, one dependency. The Go structs remain the source of truth. No schema layer, no framework lock-in. If we outgrow it, the generated types are plain interfaces — easy to migrate.

**UI test runner:**
1. **Jest** — legacy, slower, worse ESM support
2. **Vitest** — native Vite integration, fast, same config

**Chosen: Vitest.** Already using Vite, zero config friction.

**UI browser E2E:**
Considered Playwright tests against a mock Go server. Rejected — the mock server recreates the type-drift problem phase 1 eliminates. The E2E smoke test (phase 5) already proves the full stack works against the real backend.

## Applicable Skills

- `testing` — for Go test patterns and fixture conventions
- `react-best-practices` — for component test patterns
- `ts-best-practices` — for type safety in generated code
- `go-best-practices` — for Go codegen patterns

## Phases

Phases 2-3 depend on phase 1 (generated types). Phase 4 is independent and can run in parallel with 1-3.

1. [[plans/73-testing-strategy/phase-01-typesafe-api-boundary]] — Generate TS types from Go structs, delete hand-maintained types
2. [[plans/73-testing-strategy/phase-02-ui-unit-test-infrastructure]] — Install Vitest + Testing Library, configure, write first tests (depends on phase 1)
3. [[plans/73-testing-strategy/phase-03-ui-component-tests]] — Test critical component logic (Board, cards, data layer, SSE)
4. [[plans/73-testing-strategy/phase-04-expand-go-fixture-coverage]] — New fixtures for snapshot builder and loop edge cases
5. [[plans/73-testing-strategy/phase-05-e2e-agent-smoke-test]] — Real Codex agent smoke test with milestone polling

## Verification

- `go test ./...` — all Go tests pass
- `go vet ./...` — no vet issues
- `sh scripts/lint-arch.sh` — arch lint passes
- `pnpm --filter ui test` — UI unit tests pass
- `pnpm test:smoke` — E2E agent test passes (requires Codex API key, manual only)
- `pnpm fixtures:hash` — all fixture hashes current
