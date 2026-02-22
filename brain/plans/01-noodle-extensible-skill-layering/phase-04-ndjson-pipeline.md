Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 4 — NDJSON Pipeline (Stamp + Parse)

## Goal

Build the data pipeline: raw Claude/Codex NDJSON output → timestamped lines → canonical events.

**Reference codebase:** The previous implementation has proven parse and stamp code worth consulting. Read `.noodle/reference-path` for the location, then look at `parse/` and `stamp.go` there. Adapt the patterns for the new architecture — don't copy verbatim.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"

- **End-of-phase Claude review (required).** After implementing this phase, run a non-interactive Claude review of your changes and capture NDJSON output, for example: `claude -p --output-format stream-json --verbose --include-partial-messages "Review the changes for this phase. Report risks, regressions, and missing tests." | tee .noodle/reviews/<phase-id>-review.ndjson`.
- **Observe NDJSON liveness while it runs.** Watch the review log (`tail -f .noodle/reviews/<phase-id>-review.ndjson`). Any appended NDJSON line (`stream_event`, `assistant`, `user`, `system`, `result`) means Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when no new NDJSON lines appear for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until a terminal `result` event is present in the review log and blocking findings are addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`parse/`** — Canonical event types, log adapter interface (one per provider), Claude adapter, Codex adapter. The adapter registry pattern allows adding new providers without touching the core pipeline.
- **`stamp/`** — NDJSON line processor. Reads stdin line-by-line, injects a `_ts` timestamp field into each JSON object, detects the provider adapter, parses lines to canonical events, and emits sidecar events.
- **`cmd_stamp.go`** — `noodle stamp` CLI command. Reads stdin, writes timestamped NDJSON to output file, emits sidecar events.
- **`command_catalog.go`** — Register `stamp` command.

## Data Structures

- `CanonicalEvent` — Unified event type across providers. Fields: type, message, timestamp, cost, tokens in/out
- `LogAdapter` — Interface: `Parse(line []byte) ([]CanonicalEvent, error)`. One implementation per provider.
- `StampProcessor` — Reads lines, injects timestamps, calls adapter, emits events

## Verification

### Static
- `go test ./parse/...` — Adapter tests for Claude and Codex NDJSON formats
- `go test ./stamp/...` — Processor tests (timestamp injection, adapter detection, event emission)
- All canonical event types serialize/deserialize correctly

### Runtime
- Pipe sample Claude/Codex (~/.claude, ~/.codex) NDJSON through `noodle stamp` and verify:
  - Every line has a `_ts` field
  - Canonical events are extracted correctly
  - Output file is valid NDJSON
