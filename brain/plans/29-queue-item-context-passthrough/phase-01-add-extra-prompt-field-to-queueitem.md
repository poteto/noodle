Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 1: Add extra_prompt field to QueueItem

## Goal

Add the `ExtraPrompt` field to both QueueItem struct definitions so the prioritize agent can write it and the loop can read it. Include a truncation guardrail in `NormalizeAndValidate()` so oversized values are silently capped rather than rejected. Also add the schemadoc FieldDoc entry so `noodle schema queue` stays valid immediately.

## Changes

### Add field to `internal/queuex/queue.go`

Add `ExtraPrompt string` with JSON tag `json:"extra_prompt,omitempty"` to the `Item` struct, after the `Rationale` field.

### Add field to `loop/types.go`

Add `ExtraPrompt string` with JSON tag `json:"extra_prompt,omitempty"` to the `QueueItem` struct, after the `Rationale` field.

### Add truncation in `internal/queuex/queue.go`

In `NormalizeAndValidate()`, after the existing validation checks for each item, add a truncation step: if `len([]rune(items[i].ExtraPrompt)) > 1000`, truncate to 1000 runes and set `changed = true`. Use `[]rune` conversion to avoid breaking multi-byte UTF-8 characters. Truncate at a word boundary if possible (find the last space before the limit); if the last space is before 80% of the limit (i.e. before rune index 800), hard-truncate at the rune boundary instead. Truncation is silent — no log warning, no error. This is a soft cap.

### Update `fromQueueX()` / `toQueueX()` in `loop/queue.go`

Map `ExtraPrompt` between the queuex and loop types (same pattern as the other fields).

### Add FieldDoc entry in `internal/schemadoc/specs.go`

Add a FieldDoc entry for `items[].extra_prompt` in the queue target's `FieldDocs` map:

```
"items[].extra_prompt": {Description: "supplemental instructions for the cook on how to approach the task (max ~1000 chars)"},
```

Place it after `items[].rationale` to match struct field order. This must land in the same phase as the struct field to keep `noodle schema queue` validation passing.

## Data structures

- `queuex.Item` — gains `ExtraPrompt string` field
- `loop.QueueItem` — gains `ExtraPrompt string` field

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical field addition across two structs, one mapping function, one validation clause |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Write a `queue-next.json` with `extra_prompt` populated, verify the loop reads it into the struct without error
- Write a `queue-next.json` with `extra_prompt` over 1000 chars, verify it's truncated after `NormalizeAndValidate()`
- Write a `queue-next.json` with `extra_prompt` empty or absent, verify omitempty keeps JSON clean
