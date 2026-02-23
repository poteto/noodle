# Noodle Commands (Live Introspection)

Do not maintain a hand-written command list here.

Use the CLI as the source of truth:

- Full catalog:
  - `noodle --help`
- Single command help:
  - `noodle <command> --help`
- Subcommand help:
  - `noodle <command> <subcommand> --help`

Notes:
- Noodle uses cobra for CLI. All commands support `--help` for usage, flags, and subcommands.
- For detailed flag semantics, run command help directly:
  - `noodle <command> --help`
