Back to [[plans/73-testing-strategy/overview]]

# Phase 1: Typesafe API Boundary

Subsumes todo 71.

## Goal

Eliminate the manually-mirrored `ui/src/client/types.ts` by generating TypeScript interfaces directly from Go structs. After this phase, adding a field to a Go snapshot type automatically appears in the TS client — no manual sync, no nil-vs-empty-array drift.

## Changes

**`internal/snapshot/types.go`** — Add `//go:generate` directive that runs tygo directly (no wrapper script).

**`tygo.yaml` (new, project root)** — Tygo config: packages to generate from (`internal/snapshot`, `loop` for `PendingReviewItem`), output path `ui/src/client/generated-types.ts`, type mappings (`time.Time` → `string`, `json.RawMessage` → `unknown`). Both packages must produce output into a single file or two files that are re-exported.

**`ui/src/client/types.ts`** — Delete all interface definitions that mirror Go structs (`Snapshot`, `Session`, `Order`, `Stage`, `EventLine`, `FeedEvent`, `PendingReviewItem`, `DiffResponse`). Import them from `generated-types.ts` and re-export. Keep `deriveKanbanColumns()`, `KanbanColumns`, and TS-only control types (`ControlCommand`, `ControlAck`, `ConfigDefaults`, `ControlAction`).

**`ui/src/client/generated-types.ts` (new, generated)** — Auto-generated TS interfaces. Committed to repo so UI builds without `go generate`.

**`ui/src/client/enums.ts` (new)** — Hand-written union types that tygo cannot generate: `LoopState`, `Health`, `TraceFilter` (derived from Go constants), plus `StageStatus`, `OrderStatus` (UI-only, no Go constants). These narrow string unions are more type-safe than the `string` tygo would emit. Generated interfaces reference these via tygo `type_mappings` in `tygo.yaml` (e.g., `TraceFilter: TraceFilter` with an import from `enums.ts`).

**`package.json`** — Add `"generate:types": "go generate ./internal/snapshot/..."`. Add freshness check to `pnpm check`: `go generate ./internal/snapshot/... && git diff --exit-code ui/src/client/generated-types.ts`.

## Known Limitations

- **Union/enum types** — tygo generates `string` for Go string constants, not narrow unions. `enums.ts` compensates for this. If we later adopt a schema-first approach (protobuf/ConnectRPC), this problem goes away entirely.
- **Cross-package types** — `Snapshot` references `loop.PendingReviewItem`. Tygo config must list both packages with fully-qualified module paths: `github.com/poteto/noodle/internal/snapshot` and `github.com/poteto/noodle/loop`.
- **ControlCommand/ControlAck/ConfigDefaults** — These mirror Go types in `server/` but aren't part of the snapshot package. They stay hand-maintained for now. They change infrequently and the plan acknowledges this as a remaining drift risk.

## Data Structures

- `tygo.yaml` config — packages list, output path, type map
- `enums.ts` — narrow union types for string constants
- No new Go types — pure codegen

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical codegen setup, clear spec |

## Verification

### Static
- `go generate ./internal/snapshot/...` produces valid TS
- Generated TS compiles: `pnpm --filter ui tsc --noEmit`
- Freshness check in `pnpm check`: `git diff --exit-code ui/src/client/generated-types.ts` after generate
- No hand-written type definitions remain in `types.ts` for generated types
- `go vet ./...` passes

### Runtime
- Start dev server, verify UI loads and renders snapshot data correctly
- Add a dummy field to a Go struct, run `go generate`, verify it appears in TS
- Remove the dummy field, verify TS compile fails if anything references it
- Existing `server/server_test.go` tests still pass
