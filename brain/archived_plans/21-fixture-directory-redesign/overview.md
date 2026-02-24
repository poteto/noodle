---
id: 21
created: 2026-02-22
updated: 2026-02-23
status: done
---

# Fixture Directory Redesign

## Context

Our current fixtures are single markdown files with ad-hoc fenced sections. That format cannot model full runtime state cleanly (for example, per-state `noodle.toml`), and it caused real routing regressions to be hard to encode.

We are redesigning from first principles: a fixture should be a small project directory with explicit state snapshots and explicit expectations. This treats fixture state as a real filesystem boundary, not synthetic inline JSON.

## Scope

In scope:
- Replace `*.fixture.md` as the primary test fixture format with directory fixtures.
- Require consistent metadata frontmatter in each fixture’s `expected.md`.
- Support multi-state fixture flows (`state-01`, `state-02`, ...) for state-transition bugs.
- Preserve expected-failure semantics as executable assertions (run and prove failure, never skip).
- Migrate existing fixture suites: `parse`, `adapter`, `stamp`, `event`, `monitor`, `spawner`, and `loop`.
- Add high-signal fixture cases for known state bugs (including routing defaults and planning-task scheduling).
- Replace the old fixture util implementation with a clean-slate directory-native util (delete-first mindset, no fear of removal).
- Preserve previously written test intent by migrating existing fixture coverage to the new format.

Out of scope:
- New product commands or user-facing CLI features.
- Fixture caching, remote fixture registries, or golden auto-update daemons.
- Backward compatibility with legacy single-file fixtures once migration completes.

Target fixture shape:

```text
<pkg>/testdata/
  <fixture-name>/
    noodle.toml                 # optional base config for the fixture
    state-01/
      .noodle/...               # runtime/project state snapshot
      noodle.toml               # optional state override
    state-02/
      .noodle/...
    state-03/
      .noodle/...
      noodle.toml
    expected.md                 # metadata frontmatter + expectations
```

## Constraints

- Cross-platform filesystem behavior (macOS/Linux/Windows paths) must stay deterministic.
- Keep assertion logic centralized in shared fixture utilities; no package-specific parser forks.
- Keep fixture structure stable to minimize churn.
- Expected failures must execute and pass only when a mismatch/error is observed.
- `expected.md` schema version mismatches are hard failures.
- Incompatible fixture-contract changes must bump `schema_version` and update all fixtures in the same phase.
- End state removes legacy single-file fixture util code paths entirely.
- Existing fixture coverage must migrate 1:1 by case intent (or document any intentional merge/split explicitly).
- Migration can be all-at-once (no incremental compatibility shim required).

## Implementation Notes

- Execute the whole plan in one shared worktree.
- Make at least one commit per phase, and prefer multiple commits for distinct logical changes.
- For each phase, verify completion against both this overview and the phase file before marking it complete.
- Keep fixes structural and minimal; avoid speculative abstractions beyond the fixture contract and assertions needed for this redesign.

### Design Alternatives

1. Keep single-file markdown fixtures and add more sections:
- Pros: smallest immediate delta.
- Cons: still cannot represent real project state or per-state config elegantly.

2. Hybrid format (one markdown file + optional sidecar state files):
- Pros: less migration work.
- Cons: two fixture contracts create drift and confusion.

3. Directory-native fixture contract (chosen):
- Pros: explicit state boundaries, clear multi-step transitions, easier runtime realism.
- Cons: larger one-time migration.

Chosen because it most directly encodes stateful behavior and removes recurring parser brittleness.

## Applicable Skills

- `worktree`
- `debugging`
- `commit`
- `review`

## Phases

- [x] [[archived_plans/21-fixture-directory-redesign/phase-01-scaffold]] — Scaffold
- [x] [[archived_plans/21-fixture-directory-redesign/phase-02-fixture-directory-contract-and-loader]] — Fixture Directory Contract and Loader
- [x] [[archived_plans/21-fixture-directory-redesign/phase-03-metadata-and-assertion-contract]] — Metadata and Assertion Contract
- [x] [[archived_plans/21-fixture-directory-redesign/phase-04-parse-adapter-stamp-migration]] — Parse Adapter Stamp Migration
- [x] [[archived_plans/21-fixture-directory-redesign/phase-05-event-monitor-spawner-migration]] — Event Monitor Spawner Migration
- [x] [[archived_plans/21-fixture-directory-redesign/phase-06-loop-state-snapshot-model]] — Loop State Snapshot Model
- [x] [[archived_plans/21-fixture-directory-redesign/phase-07-loop-fixture-migration]] — Loop Fixture Migration
- [x] [[archived_plans/21-fixture-directory-redesign/phase-08-fixture-validator-and-tooling]] — Fixture Validator and Tooling
- [x] [[archived_plans/21-fixture-directory-redesign/phase-09-cleanup-and-final-verification]] — Cleanup and Final Verification

## Verification

Project-level static verification:
- `go test ./...`
- `go vet ./...`

Project-level runtime verification:
- Run targeted loop fixture suites that encode known bug behavior and expected-failure behavior.
- Validate at least one full multi-state fixture transition sequence end-to-end.
- Validate one fixture with per-state `noodle.toml` override to prove routing/config state is respected.
