Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 4: Tests

## Goal

Add test coverage for the new field across all touch points: struct serialization, prompt construction, and truncation. Tests go in existing test files alongside the code they exercise.

## Changes

### `loop/util_test.go` — prompt construction tests

Add table-driven tests for `buildCookPrompt()` covering:

1. **extra_prompt populated** — Verify the output contains `Scheduling context:` heading followed by the extra_prompt text, and that it appears after `Context:` (rationale) and before resume prompt
2. **extra_prompt empty** — Verify no `Scheduling context:` heading appears, and no extra blank lines
3. **extra_prompt whitespace-only** — Treated as empty, same as above
4. **extra_prompt with all other fields populated** — Verify correct ordering: header, skill, prompt, rationale, scheduling context, resume prompt

### `internal/orderx/orders_test.go` — truncation tests

Add tests in the existing normalization test section for:

1. **extra_prompt under limit** — Passes through unchanged
2. **extra_prompt at exactly 1000 chars** — Passes through unchanged
3. **extra_prompt over 1000 chars** — Truncated to ~1000 at a word boundary, `changed` returns true
4. **extra_prompt over 1000 chars with no spaces** — Hard-truncated to 1000 runes
5. **extra_prompt over limit, last space before 80% threshold** — Hard-truncated at rune boundary instead of word boundary (verifies the 80% fallback)

### `internal/orderx/orders_test.go` — round-trip serialization

Add a test that marshals an `OrdersFile` with `ExtraPrompt` set on a stage, unmarshals it, and verifies the field survives. Also verify `omitempty` behavior: marshal with empty `ExtraPrompt`, verify `extra_prompt` key is absent from JSON output.

## Data structures

No new types. Tests exercise the `ExtraPrompt` field on `orderx.Stage`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Table-driven tests with clear input/output specs |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- All new tests pass
- Existing prompt construction tests still pass (no regressions)
- `go test -cover ./loop/... ./internal/orderx/...` shows coverage on the new code paths
