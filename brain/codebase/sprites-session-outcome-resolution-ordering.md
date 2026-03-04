# Sprites Session Outcome Resolution Ordering

- `spritesSession.waitAndSync()` must run sync-back before terminal resolution so remote push/write warnings are emitted while the session is still active.
- Completion classification should use `resolveAndMarkDone(exitCode, ctxCancelled)` (event-driven) instead of manual exit-code-to-status mapping.
- Stream completion is a hard ordering barrier: resolve outcome only after canonical stream processing closes (`closeStreamDone`), or terminal events can be missed.
- Non-zero exit with observed `EventResult`/`EventComplete` should still classify as `completed`; non-zero exit with no lifecycle events should classify as `failed`.

See also [[codebase/session-outcome-interface-migration]] and [[plans/113-deterministic-completion-detection/phase-05-sprites-session-alignment]]
