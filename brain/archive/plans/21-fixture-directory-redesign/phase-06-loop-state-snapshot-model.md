Back to [[archive/plans/21-fixture-directory-redesign/overview]]

# Phase 6 — Loop State Snapshot Model

## Goal
Define the multi-state fixture model for loop tests so state transitions are explicit and inspectable.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Specify how `state-01`, `state-02`, ... are applied in sequence.
- Define where runtime state artifacts live inside each state directory (`.noodle/`, optional `noodle.toml`).
- Define expected transition assertions in `expected.md` mapped by explicit state directory key (no numeric index math in expectations).
- Ensure routing-provider/model assertions can read config from fixture state, not synthetic fields only.
- Document and enforce one mapping rule:
  - `state-01` in filesystem maps to `state-01` expectation key in `expected.md`.
  - Assertions are interpreted as full-state expectations for that state key, not edge transitions.
  - Ordering comes from lexicographic `state-XX` directory order.
- Add a small contract example showing expected mapping:
  - `state-01` -> initial spawn assertions.
  - `state-02` -> post-cycle transition assertions.

## Data Structures
- `LoopFixtureStateStep` — one ordered runtime snapshot and optional per-step config.
- `LoopTransitionExpectation` — expected loop state for each cycle step.
- `LoopRoutingExpectation` — provider/model/skill routing assertions tied to first spawn or repair spawn.

## Verification
### Static
- Loop fixture model tests cover step ordering, missing intermediate states, and config override resolution.
- Loop fixture model tests cover missing expectation keys for existing `state-XX` directories.

### Runtime
- Reproduce known routing/config bug with stateful fixture inputs and verify mismatch is captured deterministically.
