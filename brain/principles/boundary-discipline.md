# Boundary Discipline

**Principle:** Place validation, type narrowing, and error handling at system boundaries. Trust internal code unconditionally. Business logic lives in pure functions; the shell is thin and mechanical.

## Why

Validation scattered throughout a codebase is noisy, redundant, and gives a false sense of safety. Concentrating it at boundaries means each piece of data is validated exactly once — at the point it enters the system — and flows freely after that. Similarly, logic tangled with framework wiring can't be tested without the framework and can't be reused across contexts.

## The Pattern

- **At boundaries** (IPC, socket protocol, user input, external APIs): `unknown` types, explicit validation, error handling, defensive checks.
- **Inside the system**: typed data, `?` propagation, no re-validation. Trust the types.
- **Cross-language boundaries** (Rust↔TS): Generate types from a single source of truth (e.g. `ts-rs`) rather than maintaining parallel definitions.

## Applications

### Validation and Error Handling

- `unknown` over `any` at every external boundary. Cast to specific types at point of use.
- All Tauri commands return `Result<T, String>` — errors handled at the command boundary, not inside business logic.
- No `unwrap()` or `expect()` in production Rust code — propagate with `?`.
- No `as` casts except earned casts after validation.
- Store raw data at boundaries (`raw_json` in message store) — parse lazily at use-site.

### Code Organization

Business logic lives in pure functions with no framework dependencies (`(Input) => Output`). The shell — event listeners, hooks, IPC routing — is thin and mechanical, doing nothing but routing data to and from the pure core.

- **Parse functions**: Pure `(input []byte) => (State, error)` transforms. No runner or TUI dependencies.
- **Prompt construction**: `buildBrief()` — structured state in, prompt string out.
- **Scoring/assessment**: Pure `(BacklogState) => []ScoredItem` transforms. No side effects.

## The Tests

Before adding a validation check, ask: **"Is this data crossing a system boundary right now?"** If not, the validation is redundant — trust the types.

Before putting logic in a hook, event listener, or framework integration point, ask: **"Can this be a pure function that the shell just calls?"** If yes, extract it.

See also [[principles/foundational-thinking]]
