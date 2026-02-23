Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 7: Debate — Task-Type Skill with Per-Task Configurable State

## Goal

Create `.agents/skills/debate/SKILL.md` as a full task-type skill with principle grounding and per-task configurable state. Debate is a **task-type skill** — the debate agent gets scheduled as a task, evaluates the context (what's being debated, how complex it is), configures the debate parameters, and runs the structured multi-round validation. Two perspectives (reviewer + responder) alternate until consensus or max rounds.

## Current State

- No `.agents/skills/debate/` exists

## Principles

- [[principles/exhaust-the-design-space]] — debate forces exploration of alternatives before commitment
- [[principles/outcome-oriented-execution]] — debates should converge on a concrete decision, not endless discussion

## Changes

- Create `.agents/skills/debate/SKILL.md` — **use the `skill-creator` skill**
- Add Principles section
- Add convergence criteria: what counts as consensus, when to stop even without consensus
- **Non-convergence escalation** — when max rounds are reached without consensus, the verdict is written with `consensus: false` and `open_questions` in `.noodle/debates/<task-id>/verdict.json`. The prioritize skill sees the inconclusive verdict when it next reads the debate state files, and can surface it to the chef or adjust parameters. No Go-level escalation — the skill reads the files and decides.
- Add role instructions: reviewer must be specific and testable (no vague "this could be better"), responder must address each critique directly
- Add debate directory structure and file naming
- Add verdict contract with decision summary
- Include `references/debate-state-schema.md` documenting the `.noodle/debates/` directory structure, `config.json` fields, `defaults.json` format, round file naming, and `verdict.json` schema

### Per-task debate state via `.noodle/debates/<task-id>/`

Each debate gets its own directory keyed by the task ID that triggered it. Multiple debates can run concurrently without conflicts.

```
.noodle/debates/
├── 23-phase-09/
│   ├── config.json
│   ├── round-01-reviewer.md
│   ├── round-02-responder.md
│   └── verdict.json
└── 15-api-redesign/
    ├── config.json
    └── ...
```

The debate agent writes `config.json` after evaluating the task context:

- **`max_rounds`** — default 3. A straightforward task → 1-2 rounds; a major refactor → 4-5.
- **`convergence`** — what counts as consensus (e.g. "unanimous", "no-high-severity-issues")
- **`focus_areas`** — what the reviewer should focus on (e.g. "performance", "API design", "backwards compatibility")

The user can set project-wide defaults in `.noodle/debates/defaults.json`. The debate agent reads defaults first, then overrides per-task based on its judgment.

This follows the pattern of other `.noodle/` state files: Go code and skills both read/write them, giving the user control through the file system.

## Data Structures

- Defaults: `.noodle/debates/defaults.json` — `{ "max_rounds": 3, "convergence": "no-high-severity-issues", "focus_areas": [] }`
- Per-task: `.noodle/debates/<task-id>/config.json` — agent overrides per-task
- Input: the artifact to debate (plan, diff, or design document) + task context
- Output: `.noodle/debates/<task-id>/` with round files + `verdict.json` — `{ "consensus": bool, "rounds": N, "summary": "decision rationale", "open_questions": ["..."] }`

## Verification

- Static: SKILL.md has frontmatter, principles, convergence criteria, role instructions, verdict contract, per-task directory schema
- Runtime: Spawn a debate task for a completed plan phase. Confirm:
  - Debate agent evaluates task complexity and writes `config.json` with appropriate `max_rounds`
  - Round files alternate reviewer/responder
  - Reviewer critiques are specific and testable
  - Responder addresses each critique
  - verdict.json is written after each responder round
  - Debate terminates at configured max rounds (not hardcoded default)
