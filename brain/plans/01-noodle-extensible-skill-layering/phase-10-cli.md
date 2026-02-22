Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 10 — CLI Commands

## Goal

Build the remaining user-facing CLI commands. The CLI is intentionally thin — the TUI is the primary interface for monitoring and intervention (pause, drain, kill, skip, steer). The CLI covers two things the TUI can't: a quick status check from another terminal, and worktree management.

Backlog and plan management are excluded from the CLI: the adapter pattern means users interact with their backlog/plan system using whatever tools they already have (Obsidian, Linear, GitHub Issues, a text editor). Agents interact via adapter skills.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.


- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`cmd_status.go`** — `noodle status`: compact one-liner showing active cooks, queue depth, total cost, loop state (running/paused/draining). Reads from `.noodle/sessions/` state files and `.noodle/queue.json`. Works whether or not the TUI is running.
- **`cmd_worktree.go`** — Wire existing `worktree/` package: `noodle worktree create|merge|cleanup|list|prune|hook`.
- **`command_catalog.go`** — Register new commands.

### What's not here (and why)

The TUI (Phase 13) handles all intervention and detailed monitoring:

| TUI covers | Key |
|------------|-----|
| Pause/resume | `p` |
| Drain | `d` |
| Kill a cook | `k` |
| Skip queue item | `x` |
| Steer cook or sous-chef | `s` |
| Session detail + logs | `enter`, `t` |
| Cost breakdown | Dashboard |
| Queue view | `q` |

For scripting/automation, the control channel (`.noodle/control.ndjson`) is the programmatic interface — any tool can append commands directly without needing CLI wrappers.

## Data Structures

No new domain types. Commands compose existing packages (config, mise, worktree).

## Verification

### Static
- `go build ./...` — All commands compile
- `go run . commands --json` — Lists all registered commands with descriptions

### Runtime
- `noodle status` shows "no active cooks" when nothing is running
- `noodle status` with active loop shows cook count, queue depth, total cost, loop state
- `noodle worktree list` works (delegates to existing worktree package)
