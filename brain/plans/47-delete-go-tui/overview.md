---
id: 47
created: 2026-02-25
status: ready
---

# Delete Go TUI

## Context

The Bubble Tea terminal TUI was the original interface for Noodle. Plan 46 (web UI) delivered a React/TypeScript replacement — a Go HTTP server with SSE streaming, embedded in the binary. The TUI code is now dead weight: ~4,200 lines of Go across `tui/` and `tui/components/`, plus Charm dependencies (`bubbletea`, `lipgloss`, `bubbles`, `chroma`), a dedicated skill file, and integration plumbing in `cmd_start.go`.

Per subtract-before-you-add: delete it all. The one addition: a lightweight log streamer so the terminal isn't silent after the TUI is gone.

## Scope

**In scope:**
- Delete `tui/` directory (18 files + 8 component files)
- Rewrite `cmd_start.go` — remove TUI launch path, remove `--headless` flag (headless becomes the only mode), simplify `runStart` to just loop + web server
- Update `cmd_start_test.go` — remove TUI-related tests, keep loop/server tests
- Remove Charm dependencies from `go.mod` / `go.sum` (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`, `github.com/charmbracelet/x/ansi`, `github.com/alecthomas/chroma/v2`)
- Delete `.claude/skills/bubbletea-tui/` skill directory
- Add terminal log streaming to `noodle start` for operator visibility
- Update `brain/todos.md` to mark TUI items done
- Update stale TUI references in skills (`.agents/skills/noodle/SKILL.md`, `.agents/skills/testing/SKILL.md`, `generate/skill_noodle.go`) and Go comments (`loop/queue.go`, `loop/queue_audit.go`, `internal/schemadoc/specs.go`, `dispatcher/tmux_dispatcher.go`)

**Out of scope:**
- `internal/snapshot` package — shared with the web server, stays
- `loop` package — core runtime, stays
- Web UI code — untouched
- `isInteractiveTerminal()` in `main.go` — used by config repair flow, stays

## Constraints

- The `--headless` flag becomes unnecessary since there's no TUI to disable. Remove it. The web server auto-start logic (currently tied to `interactive` bool) should key off `server.enabled` config only, defaulting to true. Browser opening must still be gated on `isInteractiveTerminal()` to avoid firing in CI/cron.
- `runStartWithTUI` goes away entirely — `runStart` becomes: start loop, optionally start web server, stream logs, wait.
- No backward compatibility shims for `--headless`. Anyone using it gets a clean "unknown flag" error.
- Without the TUI, the terminal needs a log stream so operators can see what's happening. The loop already writes NDJSON events to disk and some diagnostics to stderr — tail those and print human-readable one-liners for key lifecycle events (session spawn/complete/fail, queue changes, merges, state transitions).

## Applicable skills

None — mostly deletion plus a small log streamer.

## Phases

1. [[plans/47-delete-go-tui/phase-01-delete-tui-package]]
2. [[plans/47-delete-go-tui/phase-02-rewrite-cmd-start-go-without-tui]]
3. [[plans/47-delete-go-tui/phase-03-remove-charm-dependencies-and-go-mod-tidy]]
4. [[plans/47-delete-go-tui/phase-04-delete-bubbletea-tui-skill]]
5. [[plans/47-delete-go-tui/phase-05-clean-up-brain-references-and-archive-plan]]

## Verification

```bash
go build ./...
go test ./...
go vet ./...
sh scripts/lint-arch.sh
```

After all phases: `noodle start` should launch the loop + web server, open the browser, stream lifecycle events to stderr, with no TUI code remaining.
