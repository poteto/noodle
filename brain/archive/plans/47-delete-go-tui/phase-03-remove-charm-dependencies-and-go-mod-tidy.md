Back to [[archive/plans/47-delete-go-tui/overview]]

# Phase 3 — Remove Charm dependencies and go mod tidy

## Goal

Clean up `go.mod` and `go.sum` — remove all Charm/Bubble Tea dependencies that are no longer imported anywhere.

## Changes

**`go.mod`:**
- Remove direct deps: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- Remove `github.com/alecthomas/chroma/v2` (only used in `tui/highlight.go`)
- Run `go mod tidy` to clean indirect deps (`github.com/charmbracelet/colorprofile`, `github.com/charmbracelet/ultraviolet`, `github.com/charmbracelet/x/ansi`, `github.com/charmbracelet/x/term`, `github.com/charmbracelet/x/termios`, `github.com/charmbracelet/x/windows`, etc.)

**`go.sum`:**
- Automatically cleaned by `go mod tidy`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

### Static
- `go mod tidy` exits 0
- `go build ./...` passes
- No direct `charm.land/` or `alecthomas/chroma` requires remain in `go.mod` (transitive indirect deps from other packages are fine)

### Runtime
- `go test ./...` passes
