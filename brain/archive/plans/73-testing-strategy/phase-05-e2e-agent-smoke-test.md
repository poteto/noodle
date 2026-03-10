Back to [[archive/plans/73-testing-strategy/overview]]

# Phase 5: E2E Agent Smoke Test

## Goal

A single end-to-end test that proves the full Noodle cycle works: build binary → start loop → mise gathers state → schedule agent writes orders → cook agent dispatches in worktree → session completes → worktree merges. Uses a real Codex agent against a minimal test project. Manual invocation only (not CI).

## Changes

**`e2e/smoke_test.go` (new package)** — Top-level `e2e/` package, separate from unit tests. Build-tagged `//go:build e2e` so `go test ./...` skips it by default.

Test flow:
1. **Preflight checks** — Skip with clear message if any prerequisite is missing:
   - `codex` CLI on PATH
   - `tmux` on PATH
   - `CODEX_API_KEY` or equivalent auth env var set
   - `git` available
2. Build `noodle` binary via `go build`
3. Create temp project directory with full scaffolding (modeled on `sandbox.sh` "wip" stage):
   - Git repo with initial commit on `main` branch
   - `.noodle.toml` configured for Codex (`routing.defaults.provider = "codex"`, `routing.defaults.model = "gpt-5.4"`, `autonomy = "auto"`, `concurrency.max_cooks = 1`)
   - `brain/todos.md` with a single trivial todo
   - `brain/plans/index.md`
   - Backlog adapter script that outputs the todo as a backlog item
   - Required skills copied from project: `schedule`, `execute` (at minimum)
4. Start `noodle start` as a background process
5. **Phased milestone polling** (not a single wall-clock timeout):
   - Phase A (60s): Wait for `orders.json` to appear (schedule completed)
   - Phase B (120s): Wait for a session directory to appear in `.noodle/sessions/` (cook dispatched)
   - Phase C (180s): Wait for session `meta.json` to show `status: "completed"` or `status: "merged"` (cook finished)
   - Each phase has its own deadline. Failure at any phase produces a diagnostic message naming exactly which milestone wasn't reached.
6. Assert (deterministic — `autonomy = "auto"`):
   - Session completed successfully (not failed)
   - Worktree merged to main
   - The expected file exists on main branch (e.g., `hello.txt`)
   - Session `meta.json` exists with session metadata
7. Cleanup: kill noodle process, remove temp dirs — runs in `t.Cleanup()` so it executes on both success and failure

**`e2e/helpers_test.go` (new)** — Test helpers for:
- Building the noodle binary (cached across test runs via `sync.Once`)
- Creating temp project directories with required scaffolding
- Milestone polling with per-phase deadlines and backoff
- Process cleanup (kill noodle, kill any leaked tmux sessions)

**`package.json`** — Add `"test:smoke": "go test -tags e2e -timeout 600s -count=1 ./e2e/"`. Total timeout (10 min) is larger than the sum of milestone deadlines to leave buffer for cleanup.

## Known Risks

- **Flakiness from external dependencies** — Real Codex API calls can fail due to rate limits, network issues, or model latency. The test should retry once on transient failures before reporting failure.
- **Leaked processes** — If the test crashes hard, tmux sessions and worktrees may leak. `t.Cleanup` handles normal cases; a separate `e2e/cleanup.sh` script can be provided for manual cleanup.
- **Scaffold drift** — The test project setup may diverge from what `sandbox.sh` creates. Both should be checked when the loop's startup requirements change.

## Data Structures

- Test project scaffolding — `.noodle.toml`, `brain/todos.md`, `brain/plans/index.md`, adapter script, skills
- Milestone polling — struct with phase name, deadline, and check function

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | E2E test design requires judgment — process lifecycle, milestone polling, failure modes |

## Verification

### Static
- `go vet ./e2e/...`
- Build tag prevents accidental inclusion in `go test ./...`

### Runtime
- `pnpm test:smoke` — test passes end-to-end with real Codex
- Test completes within milestone deadlines
- Test cleans up all temp directories and processes on success and failure
- Test skips cleanly when prerequisites are missing (no cryptic failures)
- Run twice in a row to verify idempotency
