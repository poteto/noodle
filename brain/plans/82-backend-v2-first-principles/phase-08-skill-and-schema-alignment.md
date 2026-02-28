# Phase 8: Skill and Schema Alignment

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Align skill guidance and schema docs with the new backend contracts so scheduler behavior and runtime contracts are unambiguous.

## Changes

- Update generated schema docs for config, status, snapshot, and orders contracts
- Update schedule skill references for mode semantics and runtime vocabulary
- Update noodle skill config/control command tables
- Remove old-contract references from skill docs
- Add before/after examples for mode/control/file contracts to reduce operator ambiguity

## Data Structures

- Schema specs for `mode`, control actions, runtime capabilities, snapshot fields
- Skill frontmatter/contract tables (documentation layer)

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Skill creation | `claude` | `claude-opus-4-6` | Contract wording and behavior clarity |

## Verification

### Static

- Schema generation and docs compile with new fields only
- No references to removed `autonomy`/`schedule.run` in skill docs
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Schedule skill reads mise and emits orders compatible with new mode/runtime contract
- UI client typecheck passes against regenerated backend schema types
- Edge cases: manual mode scheduling instructions, runtime availability hints
