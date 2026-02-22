---
name: noodle
description: Document and operate the Noodle CLI. Use when listing/explaining Noodle commands, finding command flags, or creating/editing the Noodle config file at ~/.noodle/config.toml (often misremembered as .noodlerc).
---

# Noodle

Use this skill to answer command questions about Noodle and to safely create or edit its config.

## Workflow

1. Query the CLI directly first:
   - `go run -C noodle . commands --json`
   - For one command: `go run -C noodle . commands --json --command cook`
2. If behavior looks inconsistent, verify against `noodle/main.go` and the corresponding `noodle/cmd_*.go` file.
3. Read `references/config.md` before creating or editing config.
4. If the user says `.noodlerc`, correct to `~/.noodle/config.toml` (legacy `~/.noodle/config` is still supported).

## Command Lookup

- Use `go run -C noodle . commands --json` as the source of truth.
- Use `go run -C noodle . commands --json --command \"cook log\"` for focused lookup.
- For exact flags in the current code, run `cd noodle && go run . <command> -h`.
- For cook log/status, use:
  - `cd noodle && go run . cook status`
  - `cd noodle && go run . cook log -h`

## Config Workflow

1. Confirm file path: `~/.noodle/config.toml` (project overrides: `<project>/.noodle/config.toml`).
2. Ensure directory exists: `mkdir -p ~/.noodle`.
3. If editing existing config, back it up first:
   - `cp ~/.noodle/config.toml ~/.noodle/config.toml.bak.$(date +%Y%m%d-%H%M%S)`
4. Edit key/value lines using the schema in `references/config.md`.
5. Keep TOML syntax strict:
   - Use tables for grouped config (for example `[model]`, `[entity.ceo]`, `[task.execute]`)
   - `#` starts a comment
   - Quote strings (`"safe"`, `"/path"`, `"30s"`)
   - Leave numbers and booleans unquoted (`50`, `20.0`, `true`)
6. If asked to validate, parse with Noodle’s loader by running a short Go snippet from the `noodle/` module.

## References

- `references/commands.md`
- `references/config.md`
