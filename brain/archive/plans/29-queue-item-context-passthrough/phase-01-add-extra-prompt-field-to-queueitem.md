Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 1: Add extra_prompt field to Stage

## Goal

Add an `ExtraPrompt` field to the `orderx.Stage` struct so the schedule agent can write it and the loop can read it. Include a truncation guardrail in order normalization so oversized values are silently capped rather than rejected.

## Architecture note

The original plan targeted the `QueueItem` model. Plan 49 replaced it with `Order` + `Stage` in `internal/orderx/`. `Stage` already has an `Extra map[string]json.RawMessage` field, but a dedicated `ExtraPrompt string` field is cleaner for this use case — it's always a string, always used the same way, and avoids JSON encoding overhead.

## Changes

### Add field to `internal/orderx/queue.go`

Add `ExtraPrompt string` with JSON tag `json:"extra_prompt,omitempty"` to the `Stage` struct, after the `Extra` field.

### Add truncation in order normalization

In the order normalization/validation path (`internal/orderx/orders.go` or wherever `NormalizeAndValidate` lives), add a truncation step for each stage: if `len([]rune(stage.ExtraPrompt)) > 1000`, truncate to 1000 runes and set `changed = true`. Use `[]rune` conversion to avoid breaking multi-byte UTF-8 characters. Truncate at a word boundary if possible (find the last space before the limit); if the last space is before 80% of the limit (i.e. before rune index 800), hard-truncate at the rune boundary instead. Truncation is silent — no log warning, no error. This is a soft cap.

### Add FieldDoc entry in `internal/schemadoc/specs.go`

Add a FieldDoc entry for `orders[].stages[].extra_prompt` in the orders target's `FieldDocs` map:

```
"orders[].stages[].extra_prompt": {Description: "supplemental instructions for the cook on how to approach the task (max ~1000 chars)"},
```

## Data structures

- `orderx.Stage` — gains `ExtraPrompt string` field

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical field addition to one struct, one validation clause |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Write an `orders-next.json` with `extra_prompt` populated on a stage, verify the loop reads it into the struct without error
- Write an `orders-next.json` with `extra_prompt` over 1000 chars, verify it's truncated after normalization
- Write an `orders-next.json` with `extra_prompt` empty or absent, verify omitempty keeps JSON clean
