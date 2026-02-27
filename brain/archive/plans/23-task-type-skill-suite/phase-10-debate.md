Back to [[archive/plans/23-task-type-skill-suite/overview]]

# Phase 10: Debate — Structured Debate with Per-Task State

## Goal

Create `.agents/skills/debate/SKILL.md` as a task-type skill with per-task configurable state. The debate agent evaluates context, configures parameters, and runs structured multi-round validation.

## Current State

- No `.agents/skills/debate/` exists

## Principles

- [[principles/exhaust-the-design-space]] — debate forces exploration of alternatives
- [[principles/outcome-oriented-execution]] — debates should converge on a decision

## Changes

- Create `.agents/skills/debate/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Include `references/debate-state-schema.md`
- Convergence criteria: what counts as consensus, when to stop without consensus
- **Non-convergence escalation** — verdict written with `consensus: false` and `open_questions` in `.noodle/debates/<task-id>/verdict.json`. The prioritize skill sees this and surfaces it to the chef.
- Role instructions: reviewer must be specific and testable, responder must address each critique
- Verdict contract with decision summary

### Per-task state via `.noodle/debates/<task-id>/`

Each debate gets its own directory. Multiple debates run concurrently.

```
.noodle/debates/
├── defaults.json           # Project-wide defaults
├── 23-phase-09/
│   ├── config.json          # Agent-configured per task
│   ├── round-01-reviewer.md
│   ├── round-02-responder.md
│   └── verdict.json
└── 15-api-redesign/
    └── ...
```

The agent writes `config.json` after evaluating task context:
- `max_rounds` — default 3; straightforward → 1-2, major refactor → 4-5
- `convergence` — e.g. "unanimous", "no-high-severity-issues"
- `focus_areas` — e.g. "performance", "API design"

Users set project-wide defaults in `.noodle/debates/defaults.json`.

## Data Structures

- Defaults: `.noodle/debates/defaults.json`
- Per-task: `.noodle/debates/<task-id>/config.json`, round files, `verdict.json`
- Verdict: `{ consensus, rounds, summary, open_questions }`

## Verification

- Static: SKILL.md has frontmatter, principles, convergence criteria, role instructions, verdict contract
- Static: `noodle:` frontmatter and `references/debate-state-schema.md` exist
- Runtime: Spawn a debate for a plan phase. Verify:
  - Agent writes `config.json` with appropriate `max_rounds`
  - Round files alternate reviewer/responder
  - Reviewer critiques are specific and testable
  - Verdict written after rounds
  - Terminates at configured max rounds
