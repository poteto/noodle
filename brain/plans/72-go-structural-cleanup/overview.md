---
id: 72
created: 2026-02-26
status: ready
---

# Go Structural Cleanup

## Context

A first-principles audit of the Go codebase revealed structural debt that accumulated during rapid feature development. The core loop works well, but its internal organization doesn't reflect what we'd build knowing everything we know now. This plan addresses 7 areas of structural cleanup — type duplication, god structs, file organization, and package boundaries.

No behavior changes. Every phase is a pure refactor with identical external behavior before and after.

## Scope

**In scope:**
- Unify duplicated Order/Stage types between `loop` and `internal/orderx`
- Introduce typed status enums to replace string constants
- Unify cook handle types (`cookHandle`, `pendingReviewCook`, `pendingRetryCook`)
- Decompose the 39-field `Loop` struct into cohesive sub-components
- Split the 1067-line `cook.go` by lifecycle phase
- Unexport loop-internal symbols, move `recover` to `internal/`
- Evaluate and document the `runtime`/`dispatcher` package relationship

**Out of scope:**
- Behavior changes, new features, or API changes
- Web UI or TypeScript changes
- Config schema changes
- Changes to the `dispatcher`, `server`, or `worktree` packages (beyond import path updates)

## Constraints

- Every phase must pass `go test ./... && go vet ./...` before and after
- No backward-compatibility shims — migrate callers then delete (per [[principles/migrate-callers-then-delete-legacy-apis]])
- Phases are sequenced so each builds on the previous; execute in order
- Manual implementation — no autonomous routing needed

## Applicable skills

- `go-best-practices` — for all phases
- `testing` — for verifying each phase

## Alternatives considered

**Type unification approach:** (a) Loop uses orderx types directly, (b) promote orderx types to a shared `model` package, (c) keep both but generate the conversion layer. Chose (a) because orderx already has the canonical serialization types and loop shouldn't re-declare them. A separate `model` package adds a layer without adding value.

**Loop decomposition approach:** (a) Extract sub-structs composed within Loop, (b) extract fully independent packages (e.g., `loop/completion`), (c) keep flat but just split files. Chose (a) because the sub-components share Loop's lifecycle and don't need independent testing — they're organizational, not architectural boundaries.

## Phases

- [ ] [[plans/72-go-structural-cleanup/phase-01-unify-order-types]]
- [ ] [[plans/72-go-structural-cleanup/phase-02-typed-status-enums]]
- [ ] [[plans/72-go-structural-cleanup/phase-03-unify-cook-identity-types]]
- [ ] [[plans/72-go-structural-cleanup/phase-04-decompose-loop-struct]]
- [ ] [[plans/72-go-structural-cleanup/phase-05-split-cook-go-by-lifecycle]]
- [ ] [[plans/72-go-structural-cleanup/phase-06-cleanup-exports-and-packages]]
- [ ] [[plans/72-go-structural-cleanup/phase-07-evaluate-runtime-dispatcher-layering]]

## Verification

Every phase: `go test ./... && go vet ./...`

After all phases: `sh scripts/lint-arch.sh` (if it exists), manual review that no exported API surface changed for external consumers.
