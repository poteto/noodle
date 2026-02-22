Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 1 — Project Scaffold + Core Types

## Goal

Establish the Go module at the repo root (standard Go convention), CLI entry point, command dispatch pattern, and foundational types that every subsequent phase builds on. Move `go.mod` from `src/` to the repo root and relocate the `worktree` package accordingly. After this phase, `noodle commands --json` works and all kitchen brigade types compile.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"

- **End-of-phase Claude review (required).** After implementing this phase, run a non-interactive Claude review of your changes and capture NDJSON output, for example: `claude -p --output-format stream-json --verbose --include-partial-messages "Review the changes for this phase. Report risks, regressions, and missing tests." | tee .noodle/reviews/<phase-id>-review.ndjson`.
- **Observe NDJSON liveness while it runs.** Watch the review log (`tail -f .noodle/reviews/<phase-id>-review.ndjson`). Any appended NDJSON line (`stream_event`, `assistant`, `user`, `system`, `result`) means Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when no new NDJSON lines appear for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until a terminal `result` event is present in the review log and blocking findings are addressed.

## Changes

- **`go.mod`** — Move from `src/go.mod` to repo root. Update module path.
- **`main.go`** — Minimal entry point that dispatches to a command catalog.
- **`command_catalog.go`** — Central registry of all commands. Start with a single command (`commands`) that lists available commands as JSON. Each command is a struct with name, description, category, and run function. New phases register commands here.
- **`model/`** — Core types for the kitchen brigade model. This is the foundational data layer — get these right and downstream code follows naturally.
- **`worktree/`** — Move from `src/worktree/` to repo root. Update imports.
- **`.claude/settings.json`** — Update the worktree hook command to the repo-root binary path (`go run -C $CLAUDE_PROJECT_DIR . worktree hook`) so hooks keep working after the move.

## Data Structures

- `AgentID` — String wrapper for agent identifiers
- `Cook` — A cook has an ID, provider, model, status, and optional parent (for sub-agents). No Manager/Operator/Director distinction.
- `CookStatus` — Enum: `spawning`, `running`, `completed`, `failed`, `killed`
- `Provider` — Enum: `claude`, `codex`
- `ModelPolicy` — Provider + model name + reasoning level
- `Command` — Name, description, category, run function

## Verification

### Static
- `go build ./...` succeeds from repo root
- `go vet ./...` clean
- `go test ./...` passes (model type tests + existing worktree tests)

### Runtime
- `go run . commands --json` outputs valid JSON listing the `commands` command itself
- Model types can be instantiated and serialized to JSON
- `.claude/settings.json` PreToolUse Bash hook points to `go run -C $CLAUDE_PROJECT_DIR . worktree hook` (no `old_noodle` path)
