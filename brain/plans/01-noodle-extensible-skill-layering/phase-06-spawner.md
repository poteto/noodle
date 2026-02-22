Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 6 — Spawner

## Goal

Build the Spawner interface and tmux implementation. The spawner launches cook sessions in tmux, piping output through `noodle stamp` for event extraction. This is the boundary between the Go runtime and the AI agent — everything upstream is data gathering, everything downstream is agent execution.

**Reference codebase:** The previous implementation has a working tmux spawner worth consulting. Read `.noodle/reference-path` for the location, then look at `cook/spawner.go`, `cook/tmux_spawner.go`, and `cook/tmux_session.go`. The Spawner/Session interfaces are sound — simplify SpawnRequest for the kitchen brigade model.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.


- **End-of-phase Claude review (required).** After implementing this phase, run a non-interactive Claude review of your changes and capture NDJSON output, for example: `claude -p --output-format stream-json --verbose --include-partial-messages "Review the changes for this phase. Report risks, regressions, and missing tests." | tee .noodle/reviews/<phase-id>-review.ndjson`.
- **Observe NDJSON liveness while it runs.** Watch the review log (`tail -f .noodle/reviews/<phase-id>-review.ndjson`). Any appended NDJSON line (`stream_event`, `assistant`, `user`, `system`, `result`) means Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when no new NDJSON lines appear for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until a terminal `result` event is present in the review log and blocking findings are addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`spawner/`** — New package. Spawner interface, Session interface, and TmuxSpawner implementation. SpawnRequest is simplified for the kitchen brigade model — no Phase, Target, CycleNum, EntityType, TaskClass (those were old role concepts).
- **`cmd_spawn.go`** — `noodle spawn` CLI command. Takes a prompt and spawns a cook session in tmux.
- **`command_catalog.go`** — Register `spawn` command.

## Data Structures

- `Spawner` — Interface: `Spawn(ctx, SpawnRequest) (Session, error)`
- `Session` — Interface: `ID()`, `Status()`, `Events()`, `Done()`, `TotalCost()`, `Kill()`
- `SpawnRequest` — Name, prompt, provider, model, skill (optional), reasoning level, worktree path, max turns, env vars, budget cap. The spawner reads `agents.claude_dir` / `agents.codex_dir` from config to locate the correct agent CLI for the requested provider.
- `SessionEvent` — Type, message, timestamp, rationale, cost, tokens in/out
- `TmuxSpawner` — Implementation that launches in a detached tmux session, pipes through `noodle stamp`, monitors via sidecar events

### Skill injection

When a `SpawnRequest` includes a skill, the spawner loads the `SKILL.md` and `references/` content from the resolved skill path, then injects it into the agent session based on the provider:

- **Claude** — Passes the skill as a Claude Code skill (native format).
- **Codex** — Reads `SKILL.md` content and injects it as system instructions. `references/` contents are concatenated as additional context.

This keeps skills provider-agnostic — skill authors write markdown instructions, and the spawner handles delivery.

**Size limits for Codex injection:** If a skill's `references/` directory exceeds 50KB total, the spawner truncates: it includes files in alphabetical order until the budget is exhausted, then appends a note listing the omitted files. The spawner logs a warning when truncation occurs. `SKILL.md` itself is never truncated — it's the core instructions. This prevents large reference trees from consuming the entire context budget on Codex (which has a smaller effective context than Claude).

## Verification

### Static
- `go test ./spawner/...` — Unit tests for request validation, tmux command construction
- SpawnRequest with missing required fields returns error at boundary

### Runtime
- `noodle spawn --prompt "echo hello"` launches a tmux session
- Session events flow through stamp pipeline
- `noodle spawn` with `--worktree` flag enforces linked checkout
