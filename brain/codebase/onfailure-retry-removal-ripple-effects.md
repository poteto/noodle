# OnFailure/Retry Removal Ripple Effects

- Removing `pending_retry` and `on_failure` requires updating three layers together: runtime loop code, directory fixtures under `loop/testdata/`, and legacy unit/integration tests that still construct `Order.OnFailure` or `OrderStatusFailing`.
- `readSessionStatus`, `buildAdoptedCook`, and `dropAdoptedTarget` were previously housed in `loop/cook_retry.go`; deleting retry code must keep these adopted-session helpers in a non-retry file so completion reconciliation still compiles.
- Scheduler/docs prompts and schema docs must be cleaned in tandem (`loop/schedule.go`, `internal/schemadoc/specs.go`) or stale `on_failure` guidance lingers after type removal.
