Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 6 — `noodle plugin` CLI

## Goal

Add CLI commands for managing runtime plugins: list installed, install from GitHub releases, and remove.

## Changes

**New file: `cmd_plugin.go`** (or integrated into existing CLI routing)

Three subcommands:

`noodle plugin list`
- Runs `plugin.Discover()` from phase 4
- Prints name, path, and version (from `initialize` response) for each plugin
- Shows which are configured in `.noodle.toml`

`noodle plugin install <repo>`
- Takes a GitHub repo path (e.g., `poteto/noodle-runtime-sprites`)
- Fetches latest release for current OS/arch from GitHub Releases API
- Downloads binary to `~/.noodle/plugins/noodle-runtime-{name}`
- Verifies SHA256 checksum if a checksums file is present in the release (warn if no checksums available)
- Makes it executable
- Runs `initialize` handshake to verify the plugin works

`noodle plugin remove <name>`
- Removes `~/.noodle/plugins/noodle-runtime-{name}`
- Only removes from user plugin dir (won't delete system/PATH binaries)

**New file: `cmd_plugin_test.go`**
- Test list output format
- Test install with a mock GitHub release (httptest server)
- Test remove deletes correct file
- Test remove refuses to delete non-plugin-dir binaries

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Mechanical CLI commands, clear spec |

## Verification

### Static
- `go build ./...`
- `go vet ./...`
- CLI help text is accurate

### Runtime
- `noodle plugin list` shows discovered plugins
- `noodle plugin install` with mock server downloads and installs correctly
- `noodle plugin remove` deletes the binary and subsequent `list` no longer shows it
- Install on unsupported OS/arch shows clear error
