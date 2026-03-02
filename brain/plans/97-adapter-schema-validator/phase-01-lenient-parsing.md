Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 1: Lenient Parsing with Warnings + Caller Migration

## Goal

Change `ParseBacklogItems` from hard-fail to lenient parsing **and** update all callers in the same phase so the code compiles at every commit. Bad items get skipped with a warning; valid items still return.

## Changes

### `adapter/sync.go`

- `ParseBacklogItems(ndjson string) ([]BacklogItem, []string, error)` — new signature with warnings return
- On `json.Unmarshal` error: skip the line, append warning `"line N: invalid JSON: <detail>"`
- On `validateBacklogItem` failure: skip the item, append warning with the same message format as today but as a string
- Scanner errors remain fatal (returned as error) — these indicate I/O problems, not item-level issues
- **On scanner error, return `(nil, nil, err)`.** Do not return partial items or warnings alongside a fatal error — callers should not use return values when `err != nil`.
- `validateBacklogItem` changes return from `error` to `string` (empty string = valid)

Warning format is template-controlled: `fmt.Sprintf("line %d: invalid JSON: %s", lineNumber, err.Error())`. The adapter's raw output is never included directly — only Go stdlib parse-error descriptions.

### `adapter/runner.go`

- `SyncBacklog` signature: `([]BacklogItem, []string, error)` — passes through parse warnings
- Call `ParseBacklogItems`, collect both items and warnings, return both

### `mise/builder.go`

- In `Build()`, at the `b.runner.SyncBacklog()` call (line 48): capture warnings from the new 3-value return
- Append adapter warnings to the existing `warnings` slice (which already handles "sync script missing")
- **Adapter warnings are only appended when `SyncBacklog` succeeds (err == nil).** On error, the existing early-return paths apply unchanged — no partial adapter warnings propagate.
- No change to `Brief` struct — `Warnings []string` field already exists

### `adapter/fixture_test.go`

- Update call site from `(items, err)` to `(items, warnings, err)` to match new signature
- **Migrate existing error-case fixtures:** `backlog-missing-id` currently asserts `err != nil`. After this change, it should assert `err == nil`, empty items, and `warnings` containing `"missing required field id"`. Update the fixture's `expected.md` accordingly.
- Add new fixture cases for:
  - Mixed valid/invalid items (some skip, some pass)
  - Invalid JSON line (non-JSON text in NDJSON stream)
  - Missing `id` on one item, rest valid
  - Missing `title` on one item, rest valid
  - All items invalid (returns empty items + warnings, no error)

### `adapter/runner_test.go`

- `TestParseBacklogItemsValidation` (line 54): update from `(_, err)` to `(items, warnings, err)`. Assert `err == nil`, `len(items) == 0`, and `warnings` contains `"missing required field id"` — no longer an error case.

### Fixture test data

- Update `adapter/testdata/backlog-missing-id/expected.md` from error expectation to warning expectation
- Add new fixture directories under `adapter/testdata/` for warning cases
- Each fixture: `input.ndjson` + expected items JSON + expected warnings

### Tests (from former Phase 2)

- `adapter/runner_test.go` — add or update `SyncBacklog` test to check warnings propagation
- `mise/builder_test.go` (or inline test) — verify that adapter warnings flow into `brief.Warnings`
- Test: adapter with no issues → empty warnings, same behavior as before
- Test: adapter with one bad item → `SyncBacklog` returns valid items + warnings
- Test: builder receives adapter warnings → `brief.Warnings` includes them alongside builder-level warnings

## Data structures

No new types. The return signature `([]BacklogItem, []string, error)` mirrors `Builder.Build()`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Clear spec, mechanical changes to existing code |

## Verification

### Static
- `go test ./adapter/... ./mise/...` passes
- `go vet ./adapter/... ./mise/...` clean

### Runtime
- Fixture tests cover: all-valid (no warnings), mixed valid/invalid (items + warnings), all-invalid (empty items + warnings), JSON syntax errors, missing required fields
- Existing `backlog-missing-id` fixture passes with updated expectations (warning, not error)
- `TestParseBacklogItemsValidation` passes with updated assertions
- Builder test: adapter warnings propagate into `brief.Warnings`
- Builder test: adapter with fatal error → no partial warnings in brief
