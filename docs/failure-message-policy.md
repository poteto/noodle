# Failure Message Policy

## FailureMessagePolicy

Boundary error messages in migrated plan 83 paths must describe the observed
failure state. They must not describe unmet expectations.

Policy rules:

- Describe what failed or what is missing right now.
- Keep wording concrete and short.
- Avoid expectation-style terms in boundary messages: `must`, `required`,
  `requires`, `expected`.
- Keep classification typed (`FailureClass`, `FailureRecoverability`, owner,
  scope); message text is for operators and logs, not control flow.

Message style examples:

- Use `session ID missing` instead of `session ID required`.
- Use `control action missing` instead of `action required`.
- Use `stop target missing` instead of `stop requires name`.
- Use `remove order ID missing` instead of `remove requires order ID`.

## FailureExampleSet

Representative migrated boundaries and expected typed mapping:

| Boundary | Failure-state message | Class | Recoverability | Owner | Scope |
| --- | --- | --- | --- | --- | --- |
| `POST /api/control` with empty action | `control action missing` | `backend_recoverable` | `recoverable` | `backend` | `system` |
| loop control `stop` without target | `stop target missing` | `backend_recoverable` | `recoverable` | `backend` | `system` |
| loop control `remove` without order ID | `remove order ID missing` | `backend_recoverable` | `recoverable` | `backend` | `system` |
| startup fatal config diagnostics | `fatal config diagnostics prevent start` | `backend_invariant` | `hard` | `backend` | `system` |

## Contributor Checklist

When editing migrated boundary files:

- Keep boundary message text in failure-state wording.
- Add or update tests that assert both message wording and typed failure
  mapping for representative paths.
- Run `sh scripts/lint-arch.sh` to apply the expectation-style message guard.
