# Phase 06: Docs, Skills, Brain, and Tooling Hygiene

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Remove instruction/schema drift and clean planning/knowledge surfaces after core runtime correctness work is stable.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-00-scaffold-and-gates]]

## Contract alignment note

Findings that reference orders/stage/adapter schema (`89`, `95`, `105`, `106`) should align with Phase 02's canonical contract if it has landed. If Phase 06 runs in parallel before Phase 02 completes, defer contract-dependent doc updates to a follow-up pass.

## Findings in scope

- `89-96`, `104-112`, `114-117`, `119`

## Changes

- Align docs examples to one canonical schema source.
- Resolve skill-schema conflicts and stale path/reference drift.
- Normalize brain/index/todos hygiene and naming conventions.
- Remove tracked generated docs/build artifacts from source control.

## Done when

- Docs and skills no longer present contradictory scheduler/contract examples.
- Brain planning links and naming are consistent and navigable.
- Generated artifact tracking drift is eliminated.

## Verification

### Static
- `pnpm check`
- `pnpm --filter docs build`
- `go test ./skill/...`

### Runtime
- Execute docs sample flows end-to-end.
- Validate skill authoring checks against active scheduled skills.
- Walk planning links from `brain/plans/index.md` and `brain/todos.md` without dead targets.

## Rollback

- Keep docs/skills/brain cleanup in independent commit slices.
- Revert naming/policy changes independently if they disrupt existing workflows.

## Skill requirement

- Use `skill-creator` for any changes under `.agents/skills/` or `.claude/skills/`.
