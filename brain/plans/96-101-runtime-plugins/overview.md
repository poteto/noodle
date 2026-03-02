---
id: 96-101
created: 2026-03-01
status: active
---

# Runtime Plugins

Back to [[plans/index]]

## Context

Noodle's runtime layer (process, sprites, cursor) is compiled into the binary with hardcoded config types, wiring in `loop/defaults.go`, and capability registration in `rtcap`. Users can't add custom VMs or cloud runtimes without forking Noodle.

This plan defines a plugin architecture for runtimes and extracts Sprites as the first plugin (`noodle-runtime-sprites`), proving the interface works.

## Scope

**In scope:**
- JSON-RPC over stdio plugin protocol (dispatch, kill, recover, capabilities)
- Plugin host: subprocess management, event bridge, Runtime interface adapter
- Plugin discovery (`~/.noodle/plugins/`, adjacent to binary, `$PATH`)
- Generic plugin config passthrough (`[runtime.NAME]` sections)
- `noodle plugin` CLI (list, install, remove)
- Go SDK package for writing plugins (`github.com/poteto/noodle/sdk/runtime`)
- Extract Sprites to `noodle-runtime-sprites` (separate module/repo)
- Remove Sprites from core: delete built-in code, drop `sprites-go` from `go.mod`
- Plugin authoring docs

**Out of scope:**
- Cursor dispatcher extraction (future — plan 69 lands first)
- Process runtime as plugin (it's the kernel — always built-in)
- Plugin registry/marketplace
- Plugin versioning/auto-update
- Dispatcher plugins (only runtime plugins for now — dispatchers are an internal concern)

## Constraints

- **Protocol:** JSON-RPC 2.0 over stdin/stdout. Session events as interleaved NDJSON (same format the canonical event system already uses).
- **Discovery:** Convention-based naming: `noodle-runtime-{name}` binaries.
- **Config:** `[runtime.NAME]` TOML sections passed through as raw bytes to the plugin's `initialize` RPC. No hardcoded config types for plugins in core.
- **Fallback:** Process runtime remains the universal fallback when a plugin dispatch fails (preserve existing `DispatcherFactory` behavior).
- **Cross-platform:** Plugin binaries are OS-specific. `noodle plugin install` fetches the right binary for the platform.

## Alternatives Considered

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **JSON-RPC over stdio** | Simple, language-agnostic, matches adapter pattern, reuses NDJSON events | Roll own protocol | **Chosen** — simplest path, fits Noodle's existing patterns |
| **hashicorp/go-plugin (gRPC)** | Battle-tested, typed, crash recovery, reattachment | Heavy dep, requires protobuf toolchain, fewer languages | Overkill for runtime dispatch |
| **Compiled-in registry (Caddy-style)** | Zero overhead, full type safety | Requires Go + recompilation, not installable at runtime | Doesn't serve community plugin authors |

## Applicable Skills

- `go-best-practices` — all Go phases
- `testing` — all phases with tests
- `skill-creator` — if any new skills are created during execution

## Phases

1. [[plans/96-101-runtime-plugins/phase-01-protocol-types]]
2. [[plans/96-101-runtime-plugins/phase-02-jsonrpc-host]]
3. [[plans/96-101-runtime-plugins/phase-03-session-bridge]]
4. [[plans/96-101-runtime-plugins/phase-04-discovery]]
5. [[plans/96-101-runtime-plugins/phase-05-config-passthrough]]
6. [[plans/96-101-runtime-plugins/phase-06-plugin-cli]]
7. [[plans/96-101-runtime-plugins/phase-07-plugin-sdk]]
8. [[plans/96-101-runtime-plugins/phase-08-extract-sprites]]
9. [[plans/96-101-runtime-plugins/phase-09-remove-sprites-core]]
10. [[plans/96-101-runtime-plugins/phase-10-docs]]

## Verification

```
pnpm build && pnpm check
go test ./...
go vet ./...
sh scripts/lint-arch.sh
```

End-to-end: install `noodle-runtime-sprites` plugin, configure `[runtime.sprites]`, dispatch a stage with `runtime: sprites`, verify session events stream correctly and changes sync back.
