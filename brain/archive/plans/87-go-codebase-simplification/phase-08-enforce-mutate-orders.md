Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 8: Enforce `mutateOrdersState` Usage

Split into sub-phases: API change first (requires judgment), then mechanical migration.

## Goal

`mutateOrdersState()` exists in `loop/state_orders.go` as the correct pattern for order state mutations (read → mutate → write). It's only used 3 times, while ~20 call sites manually replicate the pattern. Migrate all manual sites.

Per serialize-shared-state-mutations: centralizing the read-mutate-write through a single function makes the serialization point explicit and auditable.

## 8a: Amend `mutateOrdersState` API

### Goal

Fix the unconditional-write problem before broad migration. The current API always calls `writeOrdersState` even when the mutator makes no changes, amplifying unnecessary `orders.json` rewrites that trigger file-watch/reconcile cycles.

### Changes

**Amend the mutator contract:** change the signature so the mutator returns a `changed bool` alongside the error. Skip `writeOrdersState` when `changed` is false.

```
func (l *Loop) mutateOrdersState(mutator func(*OrdersFile) (bool, error)) error
```

Update the 3 existing callers (`cook_merge.go:71`, `state_orders.go:60`) to return `changed`.

### Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- API contract change requires judgment.

### Verification

- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean
- **No-op write test:** add a test that calls `mutateOrdersState` with a mutator returning `changed=false` and asserts `writeOrdersState` is NOT called (stat file mtime before/after, or use a write-counting mock). This proves the core optimization works.
- `go test ./...` — full suite passes

## 8b: Migrate manual mutation sites

### Goal

Build complete inventory and migrate all manual `currentOrders()+writeOrdersState()` pairs.

### Changes

Run `rg "l\.currentOrders\(\)|l\.writeOrdersState\("` to build a full inventory. Known sites beyond the 3 existing callers:
- `cook_spawn.go:228`
- `cook_merge.go:180`
- `schedule.go:278`
- `pending_review.go:152`

Each migration: wrap the mutation logic in a `func(*OrdersFile) (bool, error)` closure and pass to `mutateOrdersState`.

Sites that need read-only access or do complex multi-step mutations that don't fit the closure pattern: document why `mutateOrdersState` doesn't fit and leave them as an explicit whitelist. Gate phase completion on "no unapproved `currentOrders()+writeOrdersState()` pairs."

### Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Mechanical migration against a clear API contract.

### Verification

- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean
- Grep for `l.currentOrders()` — count should drop significantly (remaining sites documented)
- Grep for `l.writeOrdersState(` — same
- `go test ./...` — full suite passes

## Data Structures

`mutateOrdersState` signature changes to `func(func(*OrdersFile) (bool, error)) error`.
