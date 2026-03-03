# Empty Repo Startup Can Succeed Without orders.json

- In a brand-new git repo with no skills and no backlog adapter, `noodle start --once` can succeed and transition to idle without creating `.noodle/orders.json`.
- Startup still scaffolds:
  - `.noodle/` runtime directory
  - `.noodle/status.json`, `.noodle/mise.json`, `.noodle/tickets.json`, `.noodle/control.lock`
  - `brain/` files and `.noodle.toml`
- `noodle status` remains the reliable health signal in this state (`orders=0`, `loop=idle`).
- E2E coverage: `TestSmokeStartOnceWithoutSkillsInEmptyRepo` in `e2e/smoke_test.go`.
