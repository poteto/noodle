Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 2: Inject extra_prompt into cook prompt

## Goal

Update `buildCookPrompt()` to inject the extra_prompt content into the cook's prompt under a `Scheduling context:` prefix. This matches the existing `Context:` pattern used for rationale. The line is omitted entirely when extra_prompt is empty, preserving the existing prompt structure.

## Changes

### Update `buildCookPrompt()` in `loop/util.go`

Current signature: `buildCookPrompt(orderID string, stage Stage, plan []string, title string, rationale string, resumePrompt string) string`

Add `extraPrompt string` parameter (or read `stage.ExtraPrompt` directly since Stage now has the field).

After the `rationale` block and before the `resumePrompt` block, add a conditional section:

If `strings.TrimSpace(extraPrompt)` is non-empty, append `"Scheduling context: " + strings.TrimSpace(extraPrompt)` to the parts slice.

The resulting prompt order becomes:
1. Header (plan or backlog item)
2. Skill reference — conditional
3. `stage.Prompt` (task description)
4. `"Context: " + rationale` (scheduling rationale)
5. `"Scheduling context: " + extraPrompt` (supplemental instructions) — conditional
6. `resumePrompt` (resume context) — conditional

This ordering puts scheduling context after rationale but before resume context, so the cook sees "what to do" then "how to do it" then "what happened last time."

## Data structures

No new types. Reads `ExtraPrompt` from the `orderx.Stage` struct added in phase 1.

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
