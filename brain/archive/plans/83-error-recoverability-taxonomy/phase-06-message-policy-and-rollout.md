# Phase 6: Message Policy and Rollout

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Normalize high-signal boundary error messages to failure-state wording and
complete rollout checks for the new taxonomy.

## Changes

- Rewrite touched boundary messages that currently express expectations
  (`must/required/expected`) into failure-state descriptions
- Document policy and examples for future contributors
- Add regression tests for representative boundary messages and classifications

## Data structures

- `FailureMessagePolicy` (documentation-level contract)
- `FailureExampleSet`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Language policy consistency and rollout review |

## Verification

### Static

- Boundary tests assert representative message style and class mapping
- Lint/grep guard confirms no newly introduced expectation-style messages
  in migrated files
- `go test ./... && go vet ./...`

### Runtime

- Manually hit key boundaries (`start` config fatal, control parse error,
  scheduler rejection, agent start failure) and verify:
  - message describes failure state
  - typed recoverability class matches expected policy
