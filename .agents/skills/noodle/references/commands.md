# Noodle Commands (Live Introspection)

Do not maintain a hand-written command list here.

Use the CLI as the source of truth:

- Full catalog:
  - `go run -C noodle . commands --json`
- Single command:
  - `go run -C noodle . commands --json --command cook`
- Single subcommand:
  - `go run -C noodle . commands --json --command "cook log"`

Human-readable table:

- `go run -C noodle . commands`

Notes:
- `noodle commands` includes both top-level commands and subcommands.
- Each entry includes summary and usage strings that are generated from the CLI command catalog.
- For detailed flag semantics, still run command help directly:
  - `go run -C noodle . <command> -h`
