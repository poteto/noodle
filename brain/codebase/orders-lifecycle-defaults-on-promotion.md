# Orders Lifecycle Defaults On Promotion

- Root cause: `consumeOrdersNext` accepted scheduler payloads where `order.status` / `stage.status` were omitted, then `prepareOrdersForCycle` hard-failed during normalization with `order status is required`.
- Correction: lifecycle defaults are now applied in two places:
  - promotion path (`consumeOrdersNext`) defaults omitted statuses before duplicate/replacement merge logic
  - normalization path (`NormalizeAndValidateOrders`) defaults omitted statuses in persisted `orders.json` to repair previously written bad state
- Defaults:
  - missing `order.status` -> `active`
  - missing `stage.status` -> `pending`
- Missing `order.id` is now recoverable by dropping that malformed order during normalization; identity cannot be inferred safely.
- If normalization still fails (for example unknown status enums or unrecoverable invariants), the cycle no longer crashes:
  - classify as scheduler-agent recoverable mistake
  - archive current `orders.json` snapshot as `orders.json.bad.<timestamp>`
  - replace live `orders.json` with a schedule repair order
  - inject the validation issue into the next scheduler prompt so it can regenerate valid output
- Effect: compact scheduler output can omit status fields without crashing the loop cycle at `build.prepare_orders`.
