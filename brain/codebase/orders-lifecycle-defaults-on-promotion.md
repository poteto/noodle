# Orders Lifecycle Defaults On Promotion

- Root cause: `consumeOrdersNext` accepted scheduler payloads where `order.status` / `stage.status` were omitted, then `prepareOrdersForCycle` hard-failed during normalization with `order status is required`.
- Correction: lifecycle defaults are now applied in two places:
  - promotion path (`consumeOrdersNext`) defaults omitted statuses before duplicate/replacement merge logic
  - normalization path (`NormalizeAndValidateOrders`) defaults omitted statuses in persisted `orders.json` to repair previously written bad state
- Defaults:
  - missing `order.status` -> `active`
  - missing `stage.status` -> `pending`
- Missing `order.id` is now recoverable by dropping that malformed order during normalization; identity cannot be inferred safely.
- Effect: compact scheduler output can omit status fields without crashing the loop cycle at `build.prepare_orders`.
