---
id: 87
created: 2026-03-01
status: active
---

# Plan 87: Go Codebase Simplification

Back to [[plans/index]]

## Context

Static analysis of the Go codebase (~56K lines, 304 files) identified systemic duplication, long functions, dead code, and inconsistent patterns. This plan addresses all findings in a principled sequence: subtract dead code first, then lay shared foundations, then migrate callers, then decompose long functions.

## Scope

**In scope:**
- Dead code deletion (unused backend abstractions, tiny wrapper packages)
- Deduplication (cloneState, terminal checks, lookupOrderStage, PID liveness)
- New shared helpers (stringx.Normalize, jsonx read/write generics)
- Routing type drift reduction (ModelPolicy/RoutingPolicy conversion helper)
- Loop-internal helper extraction (stage failure emission, mutateOrdersState enforcement)
- Long function decomposition (6 functions over 100 lines)
- Large file splits (parsers, worktree commands)

**Out of scope:**
- Session path centralization — `filepath.Join(runtimeDir, "sessions", id)` is clear enough; abstracting it for 34 files is premature (foundational-thinking: "three similar lines of code is better than a premature abstraction")
- Restructuring the `Loop` struct's 50+ fields — high risk, unclear benefit, better done as part of a feature that needs it
- Error wrapping style changes — removing trivial `fmt.Errorf("...: %w")` wraps would reduce debuggability

## Constraints

- No backward compatibility needed (CLAUDE.md: "No backward compatibility by default")
- Per migrate-callers-then-delete-legacy-apis: when a helper is introduced, migrate all callers and delete the old pattern in the same phase
- Each phase must pass `go test ./... && go vet ./...`
- Phases are independently shippable — no cross-phase dependencies that break the build

## Alternatives Considered

**State utility consolidation (Phase 3):**
- A: Methods on status types (`OrderLifecycleStatus.IsTerminal()`, `StageLifecycleStatus.IsBusy()`) — chosen. Natural Go idiom, discoverable, zero-allocation. Critical: `IsBusy` and `IsTerminal` are distinct — `pending` is non-terminal but not busy.
- B: Package-level functions in `internal/state` — rejected. Less idiomatic, harder to discover.

**JSON helpers (Phase 5):**
- A: Generics-based `jsonx.ReadJSON[T]()` — chosen conditionally. Only proceed if audit confirms 10+ genuinely identical call sites.
- B: No jsonx package — keep inline. Acceptable if audit shows fewer than 10 identical sites.

**Stage failure emission (Phase 7):**
- A: ~~Single helper with option parameters~~ — rejected after adversarial review. Call sites diverge materially in side effects.
- B: Pure state-transition helper + source-specific orchestrators — chosen. Extracts the common core (event emission) while leaving side effects (canonical events, scheduler forwarding, cleanup) at the call site.

**Routing type unification (Phase 6):**
- A: ~~Merge into single type with dual tags~~ — rejected. Violates boundary discipline; config evolution would mutate mise.json API.
- B: Keep separate types, add explicit conversion helper — chosen. Reduces drift risk without coupling boundaries.

**mutateOrdersState (Phase 8):**
- A: ~~Migrate to existing API as-is~~ — rejected. The API always writes even on no-op, amplifying unnecessary file-watch cycles.
- B: Amend API to return `changed bool`, then migrate — chosen.

## Adversarial Review

Plan reviewed by 3 Codex reviewers (Skeptic, Architect, Minimalist). **Verdict: CONTESTED** — 3 high-severity findings addressed:
1. Phase 1 `SyncResult` is live production code — narrowed deletion scope
2. Phase 3 busy/terminal conflation — added distinct `IsBusy()` method
3. Phase 8 unconditional writes — amended `mutateOrdersState` API

7 medium-severity findings incorporated across phases 4-8, 10, 12.

## Applicable Skills

- `go-best-practices` — all phases
- `testing` — all phases (verification)
- `simplify` — invoke after each phase to catch further opportunities

## Phases

Sequenced per subtract-before-you-add → foundational-thinking (scaffold) → migrate-callers-then-delete-legacy-apis:

### Subtraction (delete before building)
1. [[plans/87-go-codebase-simplification/phase-01-delete-dead-code]]
2. [[plans/87-go-codebase-simplification/phase-02-merge-tiny-packages]]

### Foundation (shared types and helpers)
3. [[plans/87-go-codebase-simplification/phase-03-consolidate-state-utilities]]
4. [[plans/87-go-codebase-simplification/phase-04-stringx-normalize]]
5. [[plans/87-go-codebase-simplification/phase-05-jsonx-helpers]]
6. [[plans/87-go-codebase-simplification/phase-06-unify-routing-types]]

### Loop internals (extract helpers, enforce patterns)
7. [[plans/87-go-codebase-simplification/phase-07-loop-event-helpers]]
8. [[plans/87-go-codebase-simplification/phase-08-enforce-mutate-orders]]
9. [[plans/87-go-codebase-simplification/phase-09-decompose-pipeline-control]]
10. [[plans/87-go-codebase-simplification/phase-10-decompose-reconcile-completion]]

### Remaining decomposition
11. [[plans/87-go-codebase-simplification/phase-11-decompose-dispatchers]]
12. [[plans/87-go-codebase-simplification/phase-12-split-large-files]]

## Verification

Every phase: `go test ./... && go vet ./...`

Full suite after final phase: `sh scripts/lint-arch.sh` (if present)
