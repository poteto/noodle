# Doctor

Troubleshoot broken Noodle runtime state.

Checklist:
- Inspect `.noodle/mise.json`, `.noodle/queue.json`, `.noodle/tickets.json`.
- Validate session `meta.json` and event files.
- Clear stale runtime artifacts only when safe.
- If issue appears to be a binary bug, prepare a reproducible report.
