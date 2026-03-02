# Steer ID Correlation And Chat Visibility

- Symptom: A single UI steer could be delivered twice.
- Root cause: WebSocket control acks were correlated by command `id`, but server decoding dropped client IDs and generated new IDs (`web-*`), so the client timed out and retried over REST.
- Fix: preserve optional `id` in `/api/control` and WS control parsing; only synthesize IDs when missing. Client now generates one ID per send attempt and reuses it across WS + REST fallback.

- Symptom: Steer prompts were not visible in session chat.
- Root cause: steer text was sent to controllers but not written to session event logs.
- Fix: record steer prompts as session `action` events with payload `{ tool: "user", summary: <prompt> }`; snapshot formatting maps `tool=user` to label `User`.

- Symptom: Steer prompts appeared only after page refresh.
- Root cause: WS client deduped live `session_event` updates by timestamp-only and dropped legitimate out-of-order events.
- Fix: dedupe by full event identity (`at`, `label`, `body`, `category`) and keep the list time-sorted after append.

- Scheduler-specific behavior: when target is `schedule` and the active scheduler controller is not steerable, avoid kill+respawn (which looked like a misleading spawn); record the prompt and update schedule rationale for the next run.

See also [[codebase/scheduler-steer-alias-prefers-live-session]], [[principles/fix-root-causes]]
