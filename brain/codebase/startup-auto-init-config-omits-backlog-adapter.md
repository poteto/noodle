# Startup Auto-Init Config Omits Backlog Adapter

- Running `noodle start --once` in a repo without `.noodle.toml` triggers onboarding auto-init.
- The generated config can omit `[adapters.backlog]`, which means backlog sync scripts are not executed.
- E2E tests that need backlog sync behavior must write an explicit `.noodle.toml` with `adapters.backlog.scripts.sync`.

See also [[codebase/root-tests-require-ui-dist-assets]]
