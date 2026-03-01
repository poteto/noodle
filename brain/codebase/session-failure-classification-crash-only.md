# Session Failure Classification: Crash Only

- Canonical `error` events are telemetry, not terminal status signals.
- Session status should only be `failed` when runtime evidence suggests a crash (for process runtime: signal exit or no canonical lifecycle evidence before non-zero exit).
- Non-zero exits with canonical lifecycle events (`init`/`action`/`error`/`delta`) should resolve to `completed` unless a terminal completion event already exists.
- Monitor status derivation should treat `alive=false + has_events=true` as `exited` when no explicit failure claim exists.
- Avoid mapping canonical `error` directly to `state_change -> failed`; publish as action/error context instead.
