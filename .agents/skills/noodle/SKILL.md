---
name: noodle
description: Document and operate the Noodle CLI. Use when listing/explaining Noodle commands, finding command flags, or creating/editing the Noodle config file at noodle.toml (project root).
---

# Noodle

Use this skill to answer command questions about Noodle and to safely create or edit its config.

## Workflow

1. Query the CLI directly first:
   - `noodle commands --json`
   - For one command: `noodle commands --json --command cook`
2. If behavior looks inconsistent, verify against `main.go` and the corresponding `cmd_*.go` file.
3. Read `references/config.md` before creating or editing config.
4. If the user says `.noodlerc`, correct to `noodle.toml` (project root).

## Command Lookup

- Use `noodle commands --json` as the source of truth.
- Use `noodle commands --json --command "cook log"` for focused lookup.
- For exact flags in the current code, run `noodle <command> -h`.

## Config Workflow

1. Confirm file path: `noodle.toml` in the project root.
2. If editing existing config, back it up first:
   - `cp noodle.toml noodle.toml.bak.$(date +%Y%m%d-%H%M%S)`
3. Edit key/value lines using the schema in `references/config.md`.
4. Keep TOML syntax strict:
   - Use tables for grouped config (for example `[routing]`, `[prioritize]`, `[concurrency]`)
   - `#` starts a comment
   - Quote strings (`"safe"`, `"/path"`, `"30s"`)
   - Leave numbers and booleans unquoted (`50`, `20.0`, `true`)
5. If asked to validate, run `noodle start` — it reports diagnostics.

## References

- `references/config.md`
