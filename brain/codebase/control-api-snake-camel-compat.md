# Control API Snake/Camel Compat

- `/api/control` canonical JSON is snake_case (for example `order_id`).
- UI/client payloads may arrive as camelCase (`orderId`) depending on build/version skew.
- Server should normalize both when decoding control commands so merge/reject/request-changes do not fail with `order ID empty`.

See also [[principles/fix-root-causes]], [[codebase/claude-code-ndjson-protocol]]
