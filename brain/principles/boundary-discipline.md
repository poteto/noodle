# Boundary Discipline

**Principle:** Place validation, type narrowing, and error handling at system boundaries. Trust internal code unconditionally. Business logic lives in pure functions; the shell is thin and mechanical.

## Why

Validation scattered throughout a codebase is noisy, redundant, and gives a false sense of safety. Concentrating it at boundaries means each piece of data is validated exactly once — at the point it enters the system — and flows freely after that. Similarly, logic tangled with framework wiring can't be tested without the framework and can't be reused across contexts.

## The Pattern

- **At boundaries** (CLI args, TOML config, external APIs, NDJSON protocol): validate, return `error`, handle defensively.
- **Inside the system**: typed data, `return err` propagation, no re-validation. Trust the types.

## Applications

### Validation and Error Handling

- All CLI commands return `(T, error)` — errors handled at the command boundary, not inside business logic.
- No `panic()` in production code — propagate with `return err`.
- Validate config at parse time (TOML boundary), not inside business logic.
- Store raw data at boundaries (`json.RawMessage`) — parse lazily at use-site.

### Code Organization

Business logic lives in pure functions with no framework dependencies (`(Input) => (Output, error)`). The shell — CLI routing, TUI event handling, tmux management — is thin and mechanical.

- **Parse functions**: Pure `([]byte) => (State, error)` transforms. No runner or TUI dependencies.
- **Prompt construction**: `buildBrief()` — structured state in, prompt string out.
- **Scoring/assessment**: Pure `(BacklogState) => []ScoredItem` transforms. No side effects.

## The Tests

Before adding a validation check, ask: **"Is this data crossing a system boundary right now?"** If not, the validation is redundant — trust the types.

Before putting logic in a hook, event listener, or framework integration point, ask: **"Can this be a pure function that the shell just calls?"** If yes, extract it.

See also [[principles/foundational-thinking]]
