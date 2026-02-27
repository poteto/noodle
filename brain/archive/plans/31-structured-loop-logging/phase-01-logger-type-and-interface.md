Back to [[archive/plans/31-structured-loop-logging/overview]]

# Phase 1: Logger type and interface

## Goal

Add a `Logger` field to the `Dependencies` struct so callers can inject a `*slog.Logger`. Provide a default that writes text-format structured logs to stderr. This phase only adds the field and default wiring — no log call sites yet.

## Changes

### Add Logger to Dependencies (`loop/types.go`)

Add a `Logger *slog.Logger` field to the `Dependencies` struct. This follows the same pattern as other optional dependencies (`Now`, `QueueFile`, `QueueNextFile`) — callers can provide one, or get a sensible default.

### Default logger (`loop/loop.go`)

In `New()`, after the existing nil-checks for optional deps, add:

```
if deps.Logger == nil {
    deps.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
}
```

### Store on Loop struct (`loop/types.go` or `loop/loop.go`)

Add a `logger *slog.Logger` field to the `Loop` struct. Set it from `deps.Logger` in `New()`. All future log calls use `l.logger`.

## Data structures

- `Dependencies.Logger` — `*slog.Logger`, optional, defaults to stderr text handler
- `Loop.logger` — `*slog.Logger`, set from deps

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Single field addition, mechanical wiring |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Verify `New()` with nil Logger in deps produces a Loop with a non-nil `l.logger`
- Verify `New()` with a custom Logger preserves it on `l.logger`
