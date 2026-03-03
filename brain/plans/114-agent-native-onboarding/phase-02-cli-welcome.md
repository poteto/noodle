Back to [[plans/114-agent-native-onboarding/overview]]

# Phase 2: CLI Welcome Message

## Goal

Print a concise welcome message on first `noodle start` that tells the user what just happened and what to do next.

## Changes

**`startup/firstrun.go`:**
- Remove `brain/`, `brain/index.md`, `brain/todos.md`, and `brain/principles.md` from `EnsureProjectStructure()`. The `brain/` directory is part of the brainmaxxing package, not core Noodle. `EnsureProjectStructure` should only create `.noodle/` and `.noodle.toml`.
- After `EnsureProjectStructure()` creates files, print a welcome block to the writer. Only print when files were actually created (not on subsequent runs — idempotent).
- Content: what was created, what to do next (point agent at INSTALL.md, open the web UI).
- Keep it to ~8 lines. No ASCII art, no emoji. Plain text that reads well in a terminal.
- Track whether any files were created (currently `ensureFile` prints per-file — add a boolean return or counter).
- After the file loop, if anything was created, print the welcome block.

## Data Structures

- `EnsureProjectStructure` return value may gain a `created bool` or the welcome logic can check the writer output.

## Design Notes

**brain/ removal:** Currently `EnsureProjectStructure` creates `brain/`, `brain/index.md`, `brain/todos.md`, and `brain/principles.md`. This is wrong — brain is an optional feature provided by the brainmaxxing package, not a core Noodle requirement. After this change, `EnsureProjectStructure` only creates `.noodle/` and `.noodle.toml`. The INSTALL.md flow (phase 1) seeds `todos.md` at the project root as part of the bootstrap backlog.

The welcome message is the first thing a new user sees. It should orient them without overwhelming. The key information:
1. What Noodle just created (.noodle/, .noodle.toml)
2. What they need next (point agent at INSTALL.md or write skills manually)
3. Where to look (web UI at localhost:3000, `noodle status` for CLI)

This only fires on genuine first run (when files are created). Subsequent `noodle start` runs print nothing extra. The existing per-file "created X" messages stay — the welcome block follows them.

**Known limitation:** File creation is a proxy for "first run," not a true onboarding state tracker. This is an acceptable tradeoff — the edge case is rare and a dedicated onboarding state tracker costs more than the benefit.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex` (simple Go changes against clear spec)

## Verification

### Static
- `go test ./startup/...`
- `go vet ./startup/...`
- Test: call `EnsureProjectStructure` on a fresh tempdir, assert welcome message in output
- Test: call `EnsureProjectStructure` on an already-initialized dir, assert no welcome message

### Runtime
- `rm -rf /tmp/test-project && mkdir /tmp/test-project && cd /tmp/test-project && git init && noodle start`
- Verify welcome message appears
- Run `noodle start` again, verify no welcome message
