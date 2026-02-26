---
id: 59
created: 2026-02-26
status: ready
---

# Subtract Go Logic and Resilience

Covers todos #59, #61, #62, #63, #64. (#60 is folded into #49.)

## Context

The loop has several places where Go code makes decisions that should be declarative — hardcoded bootstrap prompts, pre-filtered plan lists, hardcoded `taskType.Key == "execute"` checks. It also has two crash paths that should degrade gracefully (missing sync script, merge conflicts). This plan moves decisions to skills/frontmatter and adds resilience to the two failure paths.

## Scope

**In scope:**
- #64 — `domain_skill` frontmatter field (replace hardcoded execute checks in cook.go, control.go, queuex, and both dispatchers)
- #59 — Bootstrap as skill file (move hardcoded prompt, fix silent exhaustion)
- #61 — Simplify mise.json building (remove `schedulablePlanIDs` pre-filtering)
- #62 — Missing sync script graceful degradation
- #63 — Merge conflicts park for pending review instead of permanent failure

**Out of scope:**
- #60 — Already addressed in #49 phase 4
- #49 work orders, #48 live steering, #66 triggers
- Web UI changes (existing components handle pending review)

## Constraints

### Design alternative: where does `domain_skill` live?

- **A: Top-level frontmatter** (`Frontmatter.DomainSkill`) — available to all skills. Wrong — only task types need domain context.
- **B: Under `noodle:` block** (`NoodleMeta.DomainSkill`) — scoped to task types only. **Chosen.** Domain skill injection is a scheduling/dispatch concern, so it belongs in the task type metadata alongside `schedule` and `permissions`.
- **C: Config-level mapping** (`.noodle.toml` maps task type → domain skill) — separates the concern from the skill. Wrong — the skill itself should declare what domain context it needs.

### Design alternative: bootstrap skill storage

- **A: `//go:embed`** — always available, requires recompile to evolve. Contradicts goal.
- **B: Disk file in `.agents/skills/bootstrap/`** — evolvable without recompiling, created by first-run scaffolding. **Chosen.** Matches the "everything is a file" philosophy. If missing, the loop logs an actionable error instead of silently giving up.
- **C: Embedded default + disk override** — more resilient but more complex. Over-engineering for the bootstrap case.

## Applicable Skills

- `go-best-practices` — all Go phases
- `skill-creator` — phases 3 and 5 (skill file creation/update)
- `testing` — all phases with Go changes

## Phases

1. [[plans/59-subtract-go-logic-and-resilience/phase-01-add-domain-skill-to-frontmatter-types]]
2. [[plans/59-subtract-go-logic-and-resilience/phase-02-wire-domain-skill-through-dispatch]]
3. [[plans/59-subtract-go-logic-and-resilience/phase-03-remove-mise-json-plan-pre-filtering]] *(includes former phase 4 — schedule skill update ships atomically with Go removal)*
4. ~~[[plans/59-subtract-go-logic-and-resilience/phase-04-update-schedule-skill-for-unfiltered-plans]]~~ *(folded into phase 3)*
5. [[plans/59-subtract-go-logic-and-resilience/phase-05-create-bootstrap-skill-file]]
6. [[plans/59-subtract-go-logic-and-resilience/phase-06-wire-bootstrap-skill-dispatch-and-fix-exhaustion]]
7. [[plans/59-subtract-go-logic-and-resilience/phase-07-sync-script-graceful-degradation]]
8. [[plans/59-subtract-go-logic-and-resilience/phase-08-merge-conflicts-to-pending-review]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Manual: `noodle start --once` against a project with a schedule skill that has `domain_skill: backlog` in frontmatter. Verify domain skill is injected. Remove sync script, verify graceful degradation. Create a merge conflict, verify it parks for review.
