Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 10: Update Plan Skill — Native Commands + Model Routing

## Goal

Update `.agents/skills/plan/SKILL.md` — **use the `skill-creator` skill** — to use the new native `noodle plan` commands (from Phase 9) and add model routing recommendations to the phase file template. This closes the loop between planning and scheduling — the planner thinks about _who_ should execute each phase, not just _what_ it does.

## Current State

- `.agents/skills/plan/SKILL.md` Step 5 references `go -C noodle run . plan create` and `go -C noodle run . plan phase` — these were stale adapter references
- Phase 9 adds native `noodle plan create/done/phase-add` commands
- Phase file template has: Goal, Changes, Data Structures, Verification — no routing

## Changes

### Update scaffolding commands (Step 5)
- Replace `go -C noodle run . plan create TODO-ID slug-name` with native `noodle plan create`
- Replace `go -C noodle run . plan phase TODO-ID add "Phase Name"` with native `noodle plan phase-add`
- Update Step 7 to use native commands for index/todo updates

### Update todo commands (Step 7)
- Replace stale `go -C noodle run . todo add/edit` with the configured backlog adapter commands
- The plan skill should invoke the adapter scripts configured in `[adapters.backlog]` — not hard-coded paths
- Note: backlog stays as adapter (unlike plans which became native)

### Add Routing to phase file template

Add a **Routing** section to each phase file, after Verification:

```markdown
## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Rationale: Implementation-focused — coding against a clear spec.
```

Add routing guidance table to the plan skill so authors know what to recommend:

| Phase type | Provider | Model | When |
|------------|----------|-------|------|
| Implementation / coding | `codex` | `gpt-5.3-codex` | Writing/editing code against a clear spec |
| Architecture / judgment | `claude` | `claude-opus-4-6` | Design decisions, tradeoff analysis, complex refactoring |
| Skill creation / brain work | `claude` | `claude-opus-4-6` | Writing skills, brain notes, nuanced judgment |
| Scaffolding / mechanical | `codex` | `gpt-5.3-codex` | Boilerplate, config files, mechanical transforms |

The prioritize skill reads these routing hints when scheduling phases for execution.

### Update backlog item link-back
- After creating a plan, the plan skill should update the backlog item to link back to the plan
- Use the configured backlog adapter's edit command to add the plan wikilink to the todo description
- This closes the loop: backlog item → plan → phases → execution → backlog item marked done

## Verification

- Plan skill Step 5 uses `noodle plan create` and `noodle plan phase-add` (native commands)
- Plan skill Step 7 uses backlog adapter commands for todo operations
- Phase file template includes Routing section with provider/model/rationale
- No remaining references to `go -C noodle run . plan` or `go -C noodle run . todo`
- Plan skill updates backlog item to link back to created plan (via configured backlog adapter)
- Spawn a test plan session and verify it produces phase files with Routing sections
