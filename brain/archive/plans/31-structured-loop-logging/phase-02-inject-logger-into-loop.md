Back to [[archive/plans/31-structured-loop-logging/overview]]

# Phase 2: Inject logger into loop

## Goal

Replace the 3 existing ad-hoc `fmt.Fprintf(os.Stderr, ...)` calls with `l.logger` equivalents. After this phase, all stderr output from the loop package goes through slog.

## Changes

### Replace fmt.Fprintf in cook.go

`cook.go:347` — `fmt.Fprintf(os.Stderr, "session %s failed: %s\n", ...)` in `retryCook()`. **Remove this call without adding a replacement.** Phase 5 adds the proper structured logs for both the retry path (`"cook retrying"`) and the permanent failure path (`"cook failed permanently"`), which fully cover this call site. Adding a log here would produce duplicate lines.

### Replace fmt.Fprintf in pending_review.go

`pending_review.go:72` — `fmt.Fprintf(os.Stderr, "warning: nil entry in pendingReview map\n")` in `writePendingReview()`. Replace with:

```
l.logger.Warn("nil entry in pendingReview map")
```

### Replace fmt.Fprintf in defaults.go

`defaults.go:52` — `fmt.Fprintf(os.Stderr, "warning: sprites runtime unavailable: sprite_name not set\n")`. This is in `defaultDependencies()`, a package-level function that doesn't have access to `l.logger`. Two options:

1. Accept a `*slog.Logger` parameter in `defaultDependencies()` and pass it from `New()`.
2. Use `slog.Default()` as a one-off since this runs once at startup.

Option 1 is cleaner. **Important:** `defaultDependencies()` is called at line 22 of `New()`, before the nil-checks that set default dependencies (including the logger). The logger nil-check must be moved BEFORE the `defaultDependencies()` call. Concretely: add `if deps.Logger == nil { deps.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil)) }` before the `defaultDependencies()` call at line 22, then pass `deps.Logger` through:

```
slog.Warn("sprites runtime unavailable: sprite_name not set")
```

becomes:

```
logger.Warn("sprites runtime unavailable: sprite_name not set")
```

where `logger` is the `*slog.Logger` parameter added to `defaultDependencies()`.

## Data structures

No new types. Signature change: `defaultDependencies(projectDir, runtimeDir, noodleBin string, cfg config.Config, logger *slog.Logger)`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Three mechanical replacements |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Run with a custom slog handler that captures output. Trigger each of the 2 replacement paths:
  - Construct a pendingReview map with a nil entry → verify warning logged
  - Configure sprites runtime without sprite_name → verify warning logged
- Verify the `fmt.Fprintf` at cook.go:347 is removed (no stderr output on session failure — phase 5 adds the structured replacement)
