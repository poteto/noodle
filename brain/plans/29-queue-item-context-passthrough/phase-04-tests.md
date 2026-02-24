Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 4: Tests

## Goal

Add test coverage for the new field across all three touch points: struct serialization, prompt construction, and truncation. Tests go in existing test files alongside the code they exercise.

## Changes

### `loop/util_test.go` — prompt construction tests

Add table-driven tests for `buildCookPrompt()` covering:

1. **extra_prompt populated** — Verify the output contains `Scheduling context:` heading followed by the extra_prompt text, and that it appears after `Context:` (rationale) and before resume prompt
2. **extra_prompt empty** — Verify no `Scheduling context:` heading appears, and no extra blank lines (extends the spirit of the existing `TestBuildCookPromptWithoutPrompt`)
3. **extra_prompt whitespace-only** — Treated as empty, same as above
4. **extra_prompt with all other fields populated** — Verify correct ordering: header, prompt, rationale, scheduling context, resume prompt

### `internal/queuex/queue_test.go` — truncation tests

Add tests in the existing `NormalizeAndValidate` test section for:

1. **extra_prompt under limit** — Passes through unchanged
2. **extra_prompt at exactly 1000 chars** — Passes through unchanged
3. **extra_prompt over 1000 chars** — Truncated to ~1000 at a word boundary, `changed` returns true
4. **extra_prompt over 1000 chars with no spaces** — Hard-truncated to 1000 runes
5. **extra_prompt over limit, last space before 80% threshold** — Hard-truncated at rune boundary instead of word boundary (verifies the 80% fallback)

### `internal/queuex/queue_test.go` — round-trip serialization

Add a test that marshals a `Queue` with `ExtraPrompt` set, unmarshals it, and verifies the field survives. Also verify `omitempty` behavior: marshal with empty `ExtraPrompt`, verify `extra_prompt` key is absent from JSON output.

### `loop/queue_test.go` — `fromQueueX`/`toQueueX` round-trip

Add a test that verifies `fromQueueX(toQueueX(queue))` preserves `ExtraPrompt`. Construct a `Queue` with an item that has `ExtraPrompt` set, convert to queuex and back, and assert the field value is identical.

### `loop/loop_test.go` (or appropriate integration test file) — consume-normalize-prompt flow

Add an integration test that:
1. Writes a `queue-next.json` with `extra_prompt` populated on an item
2. Runs the consume + read + normalize pipeline
3. Asserts the `ExtraPrompt` field survives into `buildCookPrompt()` output (i.e. the final prompt string contains `Scheduling context:` followed by the expected text)

## Data structures

No new types. Tests exercise the `ExtraPrompt` field on existing structs.

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
- Existing `TestBuildCookPromptIncludesPrompt` and `TestBuildCookPromptWithoutPrompt` still pass (no regressions)
- `go test -cover ./loop/... ./internal/queuex/...` shows coverage on the new code paths
