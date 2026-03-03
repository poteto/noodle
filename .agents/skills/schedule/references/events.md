# Event Types

## Internal Events

Emitted automatically by the loop. The V2 backend uses canonical event types:

| Event type | Meaning |
|------------|---------|
| `stage_completed` | A stage finished successfully (includes order ID, stage index) |
| `stage_failed` | A stage failed (includes reason) |
| `order_completed` | All stages in an order finished — the order is done |
| `order_failed` | An order failed terminally |
| `merge_failed` | A merge failed (includes error reason) |
| `order.dropped` | An order was removed because its task type is no longer registered |
| `order.requeued` | A failed order was reset and re-queued for another attempt |
| `registry.rebuilt` | The skill registry was rebuilt (skills added or removed) |

## External Events

Users can inject custom events via `noodle event emit <type> [payload]`. These have arbitrary types like `ci.failed`, `deploy.completed`, `test.flaky`, etc. You won't know every possible type — interpret them from context and the summary string.
