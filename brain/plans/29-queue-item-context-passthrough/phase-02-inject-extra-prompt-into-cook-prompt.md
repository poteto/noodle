Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 2: Inject extra_prompt into cook prompt

## Goal

Update `buildCookPrompt()` to inject the extra_prompt content into the cook's prompt under a `Scheduling context:` prefix. This matches the existing `Context:` pattern used for rationale. The line is omitted entirely when extra_prompt is empty, preserving the existing prompt structure.

## Changes

### Update `buildCookPrompt()` in `loop/util.go`

After the `Rationale` block and before the `resumePrompt` block, add a conditional section:

If `strings.TrimSpace(item.ExtraPrompt)` is non-empty, append `"Scheduling context: " + strings.TrimSpace(item.ExtraPrompt)` to the parts slice.

The resulting prompt order becomes:
1. Header (plan or backlog item)
2. `item.Prompt` (task description)
3. `"Context: " + item.Rationale` (scheduling rationale)
4. `"Scheduling context: " + item.ExtraPrompt` (supplemental instructions) — conditional
5. `resumePrompt` (resume context) — conditional

This ordering puts scheduling context after rationale but before resume context, so the cook sees "what to do" then "how to do it" then "what happened last time."

## Data structures

No new types. Reads `ExtraPrompt` from the existing `QueueItem` struct added in phase 1.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Single function change with clear spec — add one conditional block |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Dispatch a cook with `extra_prompt` set, verify `sessions/<id>/prompt.txt` contains `Scheduling context:` followed by the extra_prompt text
- Dispatch a cook without `extra_prompt`, verify `sessions/<id>/prompt.txt` has no `Scheduling context:` line and no extra blank lines
