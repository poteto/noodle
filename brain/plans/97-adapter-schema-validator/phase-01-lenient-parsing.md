Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 1: Lenient Parsing with Warnings

## Goal

Change `ParseBacklogItems` from hard-fail to lenient parsing. Bad items get skipped with a warning; valid items still return. The function signature changes to `([]BacklogItem, []string, error)`.

## Changes

### `adapter/sync.go`

- `ParseBacklogItems(ndjson string) ([]BacklogItem, []string, error)` — new signature with warnings return
- On `json.Unmarshal` error: skip the line, append warning `"line N: invalid JSON: <detail>"`
- On `validateBacklogItem` failure: skip the item, append warning (same message format as today but as a string, not error)
- Scanner errors remain fatal (returned as error) — these indicate I/O problems, not item-level issues
- `validateBacklogItem` changes return from `error` to `string` (empty string = valid)

### `adapter/fixture_test.go`

- Update to handle the new `(items, warnings, err)` return
- Add fixture cases for:
  - Mixed valid/invalid items (some skip, some pass)
  - Invalid JSON line (non-JSON text in NDJSON stream)
  - Missing `id` on one item, rest valid
  - Missing `title` on one item, rest valid
  - All items invalid (returns empty items + warnings, no error)

### Fixture test data

- Add new fixture directories under `adapter/testdata/` for warning cases
- Each fixture: `input.ndjson` + expected items JSON + expected warnings

## Data structures

- No new types. The return signature `([]BacklogItem, []string, error)` mirrors `Builder.Build()`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Clear spec, mechanical changes to existing code |

## Verification

### Static
- `go test ./adapter/...` passes
- `go vet ./adapter/...` clean

### Runtime
- Fixture tests cover: all-valid (no warnings), mixed valid/invalid (items + warnings), all-invalid (empty items + warnings), JSON syntax errors, missing required fields
- Verify backward compatibility: existing fixture cases still pass with empty warnings
