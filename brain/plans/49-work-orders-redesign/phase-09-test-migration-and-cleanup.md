Back to [[plans/49-work-orders-redesign/overview]]

# Phase 9: Test migration and cleanup

## Goal

Migrate all test fixtures from queue format to orders format, delete old types and dead code, verify the full suite passes.

## Changes

**`loop/testdata/*/state-*/.noodle/queue.json`** → rename to `orders.json`:
- Convert fixture queue files from `{items: [...]}` to `{orders: [{stages: [...]}]}` format
- Each old queue item becomes either a stage within an order or a single-stage order, depending on the fixture's intent
- Update `expected.md` assertions to match new runtime dump shape (spawn calls reference order IDs, not item IDs)

**`loop/fixture_test.go`** — Update harness:
- Read `orders.json` instead of `queue.json` from fixture state directories
- Update runtime dump capture to report order-based state (active order IDs, stage status)

**`loop/*_test.go`** — Migrate unit tests:
- Replace `QueueItem` construction with `Order`/`Stage` construction in all test helpers
- Update `testLoopRegistry()` if needed
- Update fake assertions (dispatch call args now come from stages)

**`internal/queuex/queue_test.go`** — Add orders-specific tests, delete queue-only tests that are now dead.

**Delete dead code (subtraction first):**
- `tui/` package — still exists despite #47 marking TUI deletion complete (confirmed: directory contains components, styles, model, tests). Delete the entire directory. It references `snapshot.QueueItem` extensively.

**Delete old queue code:**
- `loop/types.go` — delete `QueueItem`, `Queue` types
- `loop/queue.go` — delete `readQueue`, `writeQueueAtomic`, `consumeQueueNext`, conversion functions
- `internal/queuex/queue.go` — delete `Item`, `Queue` types and their read/write/validate functions (keep only orders-related code)
- `internal/snapshot/types.go` — old `QueueItem` already deleted in phase 7
- `loop/util.go` — delete `findQueueItemByTarget()`

**Migrate overlooked callers:**
- `cmd_status.go` — uses `queuex.Read()` for queue depth. Switch to `queuex.ReadOrders()` and count active orders.
- `cmd_debug.go` — references queue files. Update to orders files.
- `dispatcher/preamble.go` — references `.noodle/queue.json` in a doc string. Update to `orders.json`.
- `internal/schemadoc/specs.go` — uses `queuex.Queue{}` for schema docs. Update to `queuex.OrdersFile{}` with new field documentation.
- `loop/defaults.go` — references queue constants/paths. Update to orders paths.
- `loop/builtin_bootstrap.go` — references queue file paths in bootstrap templates. Update to orders.
- `generate/skill_noodle.go` — generates noodle skill text referencing queue concepts. Update to orders terminology.

**Rename package** (optional): `internal/queuex/` → `internal/orderx/` if the package now exclusively handles orders. This is a clean rename — update all imports.

**File cleanup:**
- Delete any `.noodle/queue.json` and `.noodle/queue-next.json` references in `.gitignore`, docs, scripts

## Data structures

- No new types — this phase only deletes and migrates

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical migration and deletion. Clear spec from prior phases.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- `grep -r "QueueItem\|queue\.json\|queue-next\|QueueItemInput\|ResolveQueueItem" --include="*.go"` returns zero matches
- `grep -r "QueueItem\|queue\.json\|queue-next" --include="*.ts" --include="*.tsx"` returns zero matches
- `sh scripts/lint-arch.sh` passes

### Runtime
- `go test ./...` — full suite passes
- `NOODLE_LOOP_FIXTURE_MODE=check go test ./loop/...` — all fixtures pass with orders format
- Manual: start noodle fresh, verify full cycle works (schedule creates orders, cook dispatches, stages advance, completion cleans up)
