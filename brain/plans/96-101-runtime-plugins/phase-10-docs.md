Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 10 — Documentation

## Goal

Write the plugin authoring guide and update existing runtime docs to reflect the plugin architecture.

## Changes

**New file: `docs/guides/runtime-plugins.md`**

Plugin authoring guide covering:
- What a runtime plugin is and when to write one
- The JSON-RPC protocol: methods, request/response shapes, event streaming
- Using the Go SDK (`sdk/runtime`) — implement interface, call `Serve()`
- Writing a plugin in other languages (protocol-only, no SDK)
- Config: how `[runtime.NAME]` TOML sections are passed to the plugin
- Discovery: naming convention (`noodle-runtime-{name}`), install locations
- Testing: how to test a plugin against the host
- Publishing: GitHub Releases conventions for `noodle plugin install`

**Modified file: `docs/concepts/runtimes.md`**
- Remove "coming soon" placeholder for custom runtimes
- Replace Sprites built-in documentation with plugin installation instructions
- Document the plugin architecture: discovery, config passthrough, fallback behavior
- Add section on available plugins (sprites as the reference implementation)

**Modified file: `docs/reference/configuration.md`**
- Update `[runtime.*]` section to explain generic plugin config passthrough
- Remove hardcoded Sprites/Cursor config field documentation
- Add example for custom plugin config

**Modified file: `docs/concepts/runtimes.md`**
- Update Sprites section: "Install the Sprites runtime plugin" instead of "built-in"

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Writing, technical communication, developer UX |

## Verification

### Static
- Docs build without broken links
- All code examples in docs are syntactically valid

### Runtime
- Follow the plugin authoring guide from scratch → verify a minimal plugin works
- Follow the Sprites installation instructions → verify cloud dispatch works
