Back to [[archived_plans/47-delete-go-tui/overview]]

# Phase 5 — Clean up references and archive plan

## Goal

Update brain files, skills, and Go comments to reflect that the TUI is gone. Mark todo #47 done.

## Changes

**`brain/todos.md`:**
- Mark item 47 (Delete Go TUI) as done

**`brain/plans/47-delete-go-tui/overview.md`:**
- Set frontmatter `status: done`

**Stale TUI references in skills:**
- `.agents/skills/noodle/SKILL.md` line 31 — "TUI Reviews tab" → update to reference web UI
- `.agents/skills/testing/SKILL.md` lines 56-73 — "TUI Testing with tmux" section. Delete the entire section (no TUI to test).
- `generate/skill_noodle.go` line 242 — same "TUI Reviews tab" text. Update to match `.agents/skills/noodle/SKILL.md` change. Then run `go generate ./generate/...` or the equivalent regeneration command to sync the committed skill output.

**Stale TUI references in Go comments:**
- `loop/queue.go:112` — comment says "so the TUI can derive cooking status". Update to "so the web UI and log streamer can derive cooking status" or similar.
- `loop/queue_audit.go:15` — comment says "for TUI consumption". Update to "for log streamer and web UI consumption".
- `internal/schemadoc/specs.go:128` — description says "read by TUI and CLI". Update to "read by web UI, log streamer, and CLI".
- `dispatcher/tmux_dispatcher.go:152` — comment says "shown in TUI, debugging". Update to "shown in web UI, debugging".

**No changes to archived plans** — they're historical records.

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

### Static
- `brain/todos.md` consistent — item 47 marked done
- Plan overview status is `done`
- `grep -rn "TUI" .agents/ generate/ loop/ internal/schemadoc/ dispatcher/` returns no stale references (only archived plans and test fixtures excluded)
- `go build ./...` passes (in case generated code changed)
- `go test ./generate/...` passes (snapshot sync)
