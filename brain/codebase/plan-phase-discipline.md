# Plan Phase Discipline

- When executing a numbered plan, re-read the phase docs before major edits, not just the overview.
- Treat each phase file as an execution contract: required shape, required assertions, and explicit verification.
- Temporary red tests are acceptable during migration phases; optimize for the phase end state, then restore green at phase boundaries.
- For fixture migrations, enforce phase-06/07 invariants directly in harness code:
  - `state-XX` directories drive execution order.
  - `expected.md` assertions are keyed by exact `state-XX` IDs.
  - Config/routing assertions must come from fixture `noodle.toml` state overrides, not synthetic setup fields.
