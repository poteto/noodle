Back to [[archive/plans/01-noodle-extensible-skill-layering/overview]]

# Phase 12 — Debate System

## Goal

Build structured multi-round debates for validating plans and code. A debate alternates between reviewer and responder roles, iteratively critiquing and refining work until consensus is reached or max rounds hit. The taster skill can invoke debates as part of quality review, or agents can initiate debates directly.

**Reference codebase:** The previous implementation has a working debate system worth consulting. Read `.noodle/reference-path` for the location, then look at `cook/debate.go` (round management, verdict reading), `cook/runner_debate.go` (continuation logic), and `templates/cook/debate.md` (prompt structure).

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.
- **Runtime verification in a fake project is required.** After static checks pass, run Noodle in a fresh temporary git repo under `/tmp` and verify the phase behavior there before closing.


- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`debate/`** — New package. Debate lifecycle: create, add round, read verdict, check consensus. Debates stored as directories with round files and a verdict.
- **`skills/debate/`** — Debate skill (SKILL.md). Teaches agents the debate protocol: how to structure critique, respond to critique, and write verdicts. Use the `skill-creator` skill when writing this.
- Integration with taster skill (Phase 11) — the taster can spawn a debate to validate a cook's work.

## Data Structures

- `Debate` — ID, target (what's being debated), directory path, max rounds (default 6)
- `Round` — Round number, role (reviewer or responder), content
- `Verdict` — Consensus boolean + summary string. Written as `verdict.json` in the debate directory.
- `DebateID` — Slugified target + short hash for collision avoidance

### Debate Protocol

1. Initiator creates a debate targeting a plan, diff, or artifact
2. Odd rounds: reviewer critiques the target or previous response
3. Even rounds: responder addresses the critique
4. After each round, the responder writes `verdict.json` with `consensus: true/false`
5. If consensus reached or max rounds hit, debate ends
6. Debate directory: `brain/debates/{debate-id}/round-1.md`, `round-2.md`, ..., `verdict.json`

### Storage

Debates live in `brain/debates/` for general debates or `brain/plans/{plan-id}/debate/` for plan-specific debates. Each debate is a directory containing round files and a verdict.

## Verification

### Static
- `go test ./debate/...` — Unit tests:
  - Create debate, add rounds, read back
  - Verdict parsing (consensus true/false)
  - Max rounds enforcement
  - Debate ID generation (slugify + hash)

### Runtime
- Create a debate, write two rounds, set consensus: debate directory has correct structure
- Taster skill invokes debate on a cook's output: debate rounds generated, verdict produced
- Max rounds reached without consensus: debate ends with `consensus: false`
