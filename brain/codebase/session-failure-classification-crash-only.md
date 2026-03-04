# Session Failure Classification

- `processSession.waitForExit()` must resolve status from in-memory canonical signals, not by re-reading `canonical.ndjson` after process exit.
- Resolution must wait for stream drain first; otherwise final `result`/`complete` events can be missed.
- Canonical `result` or `complete` is deterministic completion evidence and should resolve to `completed` even when exit code is non-zero or context cancellation races.
- Cancellation applies only when no completion signal was observed.
- Signal exits with no completion signal should resolve to `killed`.
- `init`/`action` without `result`/`complete` should resolve to `failed` ("no work produced"/"no turn completed"), not `completed`.
- `error` remains telemetry context and should not directly force terminal failure.

See also [[principles/prove-it-works]], [[principles/boundary-discipline]]
