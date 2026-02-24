---
id: 42
created: 2026-02-24
status: ready
---

# Permissions-Based Approval Gate

## Context

The loop hardcodes a `runQuality` step that spawns a quality review agent after each cook, reads a verdict JSON, and gates merging. The quality skill was merged into the review skill but the Go code still has special-case quality handling everywhere: `loop/quality.go`, TUI string literals, config autonomy modes, and the `reviewEnabled` / `QueueItem.Review` flow.

This plan replaces all of that with a declarative `permissions` object in the skill's noodle frontmatter:

```yaml
noodle:
  permissions:
    merge: false  # human must approve merge
  schedule: "..."
```

`merge` defaults to `true`. Setting `merge: false` means the loop parks the completed worktree for human review instead of auto-merging. No hardcoded skill names, no special-case orchestration. The `permissions` namespace is extensible for future capabilities (branch, backlog, etc.).

## Scope

**In scope:**
- Replace `blocking` with `permissions.merge` in frontmatter spec and Go structs. Delete `blocking` entirely — scheduling exclusivity is removed as a concept
- Delete `loop/quality.go` and all `runQuality` call sites
- Remove all hardcoded `"quality"` string literals from TUI
- Simplify the 3-mode autonomy config (full/review/approve) to 2 modes
- Add "request changes" control command (spawn new agent with feedback)
- Add TUI approval flow: approve / reject / request changes with Huh text input

**Also in scope:**
- Remove verdicts as a noodle-managed concept — delete `mise.QualityVerdict`, `tui/verdict.go`, verdict loading/rendering. Verdicts are entirely implementable in userland (skills write state files, prioritize reads them)

**Out of scope:**
- Automated review scheduling (the prioritize skill handles this via `schedule` field)
- Registry-driven task type lists in TUI (follow-up improvement)
- Additional permissions beyond `merge` (future work)

## Constraints

- **Worktree:** `remove-hardcoded-quality-skill` already has one commit (quality→review merge). Continue work there.
- **No backward compat.** Per CLAUDE.md: no legacy fallbacks, no dual-path support. `blocking` is deleted, not deprecated. Old autonomy values (`full`, `review`) are rejected, not migrated.
- **Huh library** is not currently a dependency. Add it when needed in Phase 6.

## Applicable Skills

- `go-best-practices` — Go patterns, testing conventions
- `bubbletea-tui` — TUI component design
- `testing` — TDD workflow, fixture conventions

## Alternatives Considered

**A. Flat `requires_approval` bool** — Simpler but doesn't namespace well. Adding future permissions (branch, backlog) would clutter the top-level noodle frontmatter. Rejected in favor of the `permissions` object.

**B. Keep 3-mode autonomy + per-skill permissions** — Complex interaction matrix (3 modes x N permissions). Rejected: too many states.

**C. `permissions` object with `merge` defaulting to `true`** — Chosen. Extensible, reads naturally, most skills need zero config. Global `approve` mode overrides per-skill `merge: true` when the user wants to review everything.

## Phases

- [[plans/42-requires-approval-gate/phase-01-frontmatter-spec-permissions-object]]
- [[plans/42-requires-approval-gate/phase-02-remove-hardcoded-quality-from-loop]]
- [[plans/42-requires-approval-gate/phase-03-remove-hardcoded-quality-from-tui]]
- [[plans/42-requires-approval-gate/phase-04-simplify-autonomy-config]]
- [[plans/42-requires-approval-gate/phase-05-request-changes-control-command]]
- [[plans/42-requires-approval-gate/phase-06-tui-approval-flow-with-huh-text-input]]
- [[plans/42-requires-approval-gate/phase-07-update-noodle-skill-with-permissions-docs]]

## Verification

```sh
go test ./... && go vet ./... && sh scripts/lint-arch.sh
make fixtures-loop MODE=check
make fixtures-hash MODE=check
```
