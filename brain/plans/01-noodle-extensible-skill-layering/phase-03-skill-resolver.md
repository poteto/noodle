Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 3 — Skill Resolver

## Goal

Implement layered skill resolution. A skill name resolves to a directory path by searching configured paths in order. First match wins. This is the mechanism that makes skills the only extension point — users override defaults simply by placing a skill with the same name earlier in the search path.

The resolver searches paths declared in config. There is no "bundled" tier embedded in the binary — the bootstrap skill (Phase 14) installs default skills to `~/.noodle/skills/` during setup. The typical precedence is: `skills/` (project, committed) > `~/.noodle/skills/` (user/defaults).

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

- **`skill/`** — New package. Resolver logic, skill metadata reading.
- **`cmd_skills.go`** — `noodle skills list` command that shows resolved skills with their source path.
- **`command_catalog.go`** — Register `skills` subcommands.

## Data Structures

- `Resolver` — Holds ordered search paths from config. Method: `Resolve(name string) (SkillPath, error)`
- `SkillPath` — Resolved absolute path + which search path it came from
- `SkillInfo` — Name, path, source path, whether it has a SKILL.md

Skills are pure Claude Code skills — just a directory with a `SKILL.md` and optional `references/`. The resolver doesn't parse skill content; it just finds the right directory. Noodle-specific wiring (which skill handles which phase) lives in config, not in the skill.

## Verification

### Static
- `go test ./skill/...` — Resolution precedence tests:
  - Skill in earlier search path wins over same name in later path
  - Missing skill returns clear error
  - Empty search paths handled gracefully
  - Non-existent search path directories are skipped (not errors)

### Runtime
- `noodle skills list` shows skills from all configured paths with source
- Create a project-level skill that shadows a user-level one, verify `skills list` shows the project one as active
