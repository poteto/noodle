Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 8: Enforce `mutateOrdersState` Usage

## Goal

`mutateOrdersState()` exists in `loop/state_orders.go` as the correct pattern for order state mutations (read → mutate → write). It's only used 3 times, while ~20 call sites manually replicate the pattern. Migrate all manual sites.

Per serialize-shared-state-mutations: centralizing the read-mutate-write through a single function makes the serialization point explicit and auditable.

## Changes

### Step 1: Fix `mutateOrdersState` API before broad migration

The current `mutateOrdersState` unconditionally calls `writeOrdersState` even when the mutator makes no changes. Propagating this across ~20 sites amplifies unnecessary `orders.json` rewrites, which trigger file-watch/reconcile cycles.

**Amend the mutator contract:** change the signature so the mutator returns a `changed bool` alongside the error. Skip `writeOrdersState` when `changed` is false.

```
func (l *Loop) mutateOrdersState(mutator func(*OrdersFile) (bool, error)) error
```

Update the 3 existing callers (`cook_merge.go:71`, `state_orders.go:60`) to return `changed`.

### Step 2: Build complete inventory

Run `rg "l\.currentOrders\(\)|l\.writeOrdersState\("` to build a full inventory. The original list missed active sites in:
- `cook_spawn.go:228`
- `cook_merge.go:180`
- `schedule.go:278`
- `pending_review.go:152`

### Step 3: Migrate all manual sites

Migrate from the complete inventory. Each migration: wrap the mutation logic in a `func(*OrdersFile) (bool, error)` closure and pass to `mutateOrdersState`.

Sites that need read-only access or do complex multi-step mutations that don't fit the closure pattern: document why `mutateOrdersState` doesn't fit and leave them as an explicit whitelist. Gate phase completion on "no unapproved `currentOrders()+writeOrdersState()` pairs."

## Data Structures

`mutateOrdersState` signature changes to `func(func(*OrdersFile) (bool, error)) error`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Step 1 (API change) requires judgment. Steps 2-3 are mechanical but need the complete inventory first.

## Verification

### Static
- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean
- Grep for `l.currentOrders()` — count should drop significantly (remaining sites documented)
- Grep for `l.writeOrdersState(` — same

### Runtime
- `go test ./...` — full suite passes
