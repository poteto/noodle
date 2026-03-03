# Phase 01: CLI, Config, and Adapter Boundaries

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Resolve root-boundary drift across CLI/config/adapter surfaces so boundary handling is explicit, typed, and cross-platform safe.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-00-scaffold-and-gates]]

## Findings in scope

- `1-24`

## Changes

- Make command pre-run/config loading policy command-scoped and recovery-safe.
- Remove global/ambient boundary leaks (`chdir`, global seams, stdout coupling) where they obstruct deterministic command behavior.
- Consolidate config/default sources and remove legacy/tolerated boundary drift.
- Choose one canonical backlog adapter contract path and migrate fully.
- Replace shell-string adapter execution contracts with typed argv boundaries.

## Data structures

- CLI boundary context object for command-scoped dependencies.
- Typed adapter failure taxonomy for mise/adapter integration.
- Canonical config defaults source used by parser/startup/docs generation.

## Within-phase priority

- **P1 first:** `1` (malformed config blocks recovery), `2` (global chdir ambient state), `20` (silent succeed on missing target ID).
- **P2 after:** all remaining (`3-19` minus above, `21-24`).

## Done when

- CLI recovery commands are runnable despite malformed project config where appropriate.
- Adapter/mise boundary errors are typed (`errors.Is`-style) rather than string-classified.
- Exactly one authoritative adapter execution path remains.
- Config defaults have one source of truth with parity checks.

## Verification

### Static
- `go test ./... && go vet ./...`
- `pnpm check`

### Runtime
- Run command-path smoke checks for `start/reset/worktree/event/skills` against malformed and valid config states.
- Execute backlog adapter flows (`sync/add/done/edit`) and verify not-found semantics are explicit.

## Rollback

- Keep boundary refactors in small isolated commits (CLI vs config vs adapter).
- If command usability regresses, revert only affected boundary segment without touching other subsystems.
