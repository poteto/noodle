---
id: 48
created: 2026-02-25
status: draft
---

# Plan 48 — Live Agent Steering

Back to [[plans/index]]

## Context

Noodle's steer mechanism kills the agent process and respawns it with a resume prompt. This loses the agent's full conversation context (thinking, file reads, mental model) and wastes tokens re-establishing state. Both Claude Code and Codex now support live communication channels that allow interrupt + redirect without killing the process.

## Scope

**In scope:**
- Drop tmux as the process host — Noodle spawns provider processes directly as child processes with bidirectional pipes
- Replace file-redirection stdin with `--input-format stream-json` pipes for Claude Code
- Replace `codex exec` with `codex app-server --listen stdio://` for Codex control
- New dispatcher abstraction: `AgentController` with Start/Steer/Interrupt
- Rewrite `steer` control command to use interrupt + redirect instead of kill + respawn
- Migrate reconcile, monitoring, and shutdown from tmux to PID-based lifecycle
- Run stamp processor in-process (like sprites already does) instead of as a pipeline stage

**Out of scope:**
- WebSocket transport (no multi-observer use case yet)
- Claude's `can_use_tool` permission gating (stays on `bypassPermissions`)
- Changing canonical event format or parse adapters
- Mid-generation injection for Claude (interrupt-then-redirect is sufficient)
- Sprites runtime (current sprites path explicitly avoids stdin streaming — separate work)
- Cursor runtime

## Constraints

- Claude Code uses `--input-format stream-json` / `--output-format stream-json` — raw NDJSON over stdin/stdout pipes
- Codex app-server uses JSON-RPC over stdin/stdout with `--listen stdio://`
- Claude steering flow: send `control_request { subtype: "interrupt" }` → wait for turn to stop → send new `user` message
- Codex steering flow: send `turn/steer` JSON-RPC request (true mid-turn injection) or `turn/interrupt` + `turn/start`
- Codex event parsing: read rollout JSONL files from `~/.codex/sessions/` — extract path from `thread/started` notification, not filesystem watching
- Claude event parsing: pipe stdout through in-process stamp — existing `ClaudeAdapter` stays unchanged
- Child processes need `SysProcAttr{Setpgid: true}` for process group isolation (tmux gave this for free)
- Noodle crash kills all in-flight cooks (no more tmux orphan survival) — they retry on restart

### Alternatives considered

**Tmux integration (for pipe access):**
1. **FIFO bridge** — create named pipe in session directory, tmux pipeline reads from it. Preserves tmux lifecycle. Adds plumbing complexity.
2. **Sidecar socket** — Unix domain socket instead of FIFO. Bidirectional natively but more complex.
3. **Drop tmux, direct child processes** — Noodle spawns provider via `os/exec`, holds pipes directly. ~350 lines of lifecycle migration.

Chosen: **Option 3 (drop tmux)** — cleanest pipe management, unifies with sprites architecture, removes install dependency, simplifies process model. Tradeoff: lose `tmux attach` debugging (web UI replaces it) and orphan survival across crashes (retry on restart handles it).

**Transport:**
1. **WebSocket** — same protocol, extra server/port. No benefit without multi-client observation.
2. **Hook-based soft steering** — inject via `PreToolUse` hooks. Limited to tool boundaries.
3. **Bidirectional pipes** — same protocol, simpler architecture, no server.

Chosen: **bidirectional pipes**.

### Codex app-server parity matrix

Current `codex exec` flags → app-server equivalents:

| Concern | `codex exec` flag | App-server equivalent |
|---------|-------------------|----------------------|
| Approval | `--ask-for-approval never` | Set in config or `thread/new` params |
| Sandbox | `--sandbox workspace-write` | Set in config or `thread/new` params |
| Model | `--model M` | `thread/new` params `model` field |
| Working dir | Inherited `cwd` | `thread/new` params `cwd` field |
| Git check | `--skip-git-repo-check` | Config or env var |
| JSON output | `--json` | N/A (app-server always uses JSON-RPC) |

Phase 3 must verify each mapping against live `codex app-server` before implementation.

## Applicable Skills

- `go-best-practices` — lifecycle, concurrency, error handling
- `testing` — TDD, fixtures, test patterns

## Phases

1. [[plans/48-live-agent-steering/phase-1-process-manager]]
2. [[plans/48-live-agent-steering/phase-2-controller-interface]]
3. [[plans/48-live-agent-steering/phase-3-claude-pipe-transport]]
4. [[plans/48-live-agent-steering/phase-4-codex-appserver-transport]]
5. [[plans/48-live-agent-steering/phase-5-session-lifecycle-migration]]
6. [[plans/48-live-agent-steering/phase-6-steer-rewrite]]
7. [[plans/48-live-agent-steering/phase-7-cleanup]]

## Verification

- `go test ./... && go vet ./...`
- `sh scripts/lint-arch.sh`
- Manual: start noodle, spawn a cook, send steer command, verify agent keeps context and redirects without process restart
- Manual: verify both Claude and Codex providers work with steer
- Manual: kill noodle process, restart, verify cooks are re-dispatched (not adopted from tmux)
