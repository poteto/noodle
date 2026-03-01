Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 5: Create `jsonx` Read/Write Helpers

## Goal

Reduce ReadFile→Unmarshal / Marshal→WriteFileAtomic boilerplate by introducing generic helpers in a new `internal/jsonx` package. Only migrate call sites with genuinely identical semantics.

**Prerequisite:** Before creating the package, run `rg` to audit all JSON read/write sites and confirm at least 10 have truly identical semantics (read file, unmarshal, return zero on not-found). If the audit shows fewer, this phase should be dropped.

## Changes

### New package: `internal/jsonx/jsonx.go`

Three functions:
- `ReadJSON[T any](path string) (T, error)` — reads file, returns zero value + nil if not found, unmarshals otherwise
- `WriteJSON(path string, v any) error` — marshals compact JSON, writes atomically via `filex.WriteFileAtomic`
- `WriteJSONIndented(path string, v any) error` — same but with `MarshalIndent`

Error messages follow the project convention: describe the failure state, not expectations. Use the filename as context (e.g., `"unmarshal %s: %w"`).

### Migrate callers (highest-value targets):
- **`monitor/fileio.go`** — `readSessionMeta`, `writeSessionMeta` become one-liners
- **`monitor/claims.go`** — `readSpawnMetadata` simplifies
- **`mise/io.go`** — `writeBriefAtomic` becomes one-liner

**Exclude `loop/state_orders.go`** — it already delegates to `orderx.ReadOrders` and has its own mutation pattern via `mutateOrdersState`. Adding another layer of indirection provides no simplification.

Migrate only sites where the helper is a clean drop-in. Leave sites with post-processing after unmarshal.

## Data Structures

- `func ReadJSON[T any](path string) (T, error)`
- `func WriteJSON(path string, v any) error`
- `func WriteJSONIndented(path string, v any) error`

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Generic API design requires judgment on error handling semantics (what does "not found" return?).

## Verification

### Static
- `go test ./internal/jsonx/...` — new package has tests (including not-found, invalid JSON, write-read roundtrip)
- `go build ./...` — all imports resolve
- `go vet ./...` — clean

### Runtime
- `go test ./...` — full suite passes
- Manual check: `grep -r "os.ReadFile.*json\|json.Unmarshal" --include="*.go" -l` — count decreases in migrated files
