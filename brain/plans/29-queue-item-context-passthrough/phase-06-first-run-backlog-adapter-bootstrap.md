Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 6 — First-run backlog adapter bootstrap

## Goal

When `noodle start` runs and no backlog adapter is configured, prompt the user to create one. Spawn a general-purpose agent that sets up the adapter based on where the user's work lives (Linear, GitHub Issues, Notion, markdown files, etc.).

## Changes

**New diagnostic type:**
- Add a "missing backlog adapter" diagnostic to the repair system
- Current repair flow (`config/config.go`, `main.go`) only handles missing script paths for adapters that already exist in config — it doesn't handle the case where no adapter is configured at all
- Add a new diagnostic that detects "no backlog adapter in config" as a distinct condition

**`cmd_start.go` or config validation:**
- If no backlog adapter is configured (missing from `.noodle.toml` or no adapter scripts exist), detect this at startup
- Prompt the user interactively: "No backlog adapter found. Want to create one?"
- Gather context: "Where does your work live?" (Linear, GitHub Issues, Notion, markdown file, other)
- Spawn a general-purpose agent with this context to create the adapter scripts and update `.noodle.toml`
- Follow the existing config repair pattern (`promptRepairSelection` / `startRepairSession` in `main.go`)

**Idle gate safety:**
- Verify the loop doesn't go idle (dead state) when the config has no adapters and no backlog
- First-run with no adapter should prompt, not silently idle

**UX flow:**
1. User runs `noodle start`
2. No backlog adapter → prompt appears
3. User picks their work source (or provides custom description)
4. Agent spawns, creates adapter scripts, updates config
5. User re-runs `noodle start` — adapter is now configured

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

UX design and agent prompt crafting require judgment.

## Verification

### Static
- `go build ./...` and `go test ./...` pass

### Runtime
- Start Noodle with no backlog adapter configured — verify prompt appears (not silent idle)
- Select a work source — verify agent spawns and creates adapter scripts
- Re-run `noodle start` — verify adapter is detected and backlog syncs
