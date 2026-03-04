# Session Outcome Interface Migration

- `dispatcher.Session` now has `Outcome() SessionOutcome`; the legacy `Status() string` remains during migration phases.
- `sessionBase` is the single wiring point for both `processSession` and `spritesSession`; adding outcome state there updates both runtimes at once.
- `Outcome()` should return zero value while running and only gain terminal `Status` after `Done()` closes.
- Any test double implementing `dispatcher.Session` must add `Outcome()` immediately (notably dispatcher factory tests and runtime dispatcher tests).

See also [[plans/113-deterministic-completion-detection/phase-01-session-outcome-type]] and [[principles/migrate-callers-then-delete-legacy-apis]]
