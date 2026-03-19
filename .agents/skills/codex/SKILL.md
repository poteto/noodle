---
name: codex
description: >-
  Run Codex CLI (exec, resume) for parallel work, delegating tasks to another model, or background
  code generation. Handles code analysis, refactoring, and automated editing. Includes Spark vs
  standard guidance. Triggers: "codex", "run codex", "use codex to".
---

# Codex

## Profiles

- **(no profile)** — read-only analysis (no compilation, no tests)
- **`--profile edit`** — local edits, build, or test (`workspace-write`, full-auto)
- **`--profile full`** — network or broad filesystem access (`danger-full-access`, full-auto)

Use `--profile edit` whenever the task involves `go test`, `go build`, or any command that writes build artifacts. The default read-only sandbox blocks all writes including compiler temp dirs.

For complex tasks, add `--config model_reasoning_effort="xhigh"`.

## Model Selection

Default to the project model in `.codex/config.toml` (`gpt-5.4`).

Use `--model gpt-5.3-codex-spark` for speed-first interactive work:

- tight back-and-forth iteration
- quick bug fixes and targeted edits
- small-to-medium refactors or UI tweaks
- rapid prototyping where frequent redirect/interrupt is expected

Stay on default `gpt-5.4` for depth-first work:

- long autonomous runs
- broad multi-file changes with heavy reasoning
- test-heavy or verification-heavy tasks
- higher correctness/risk surfaces where deeper reasoning is worth the extra latency

If the user explicitly requests a model, follow the user's request.

## Commands

Project defaults in `.codex/config.toml`. Always include `--skip-git-repo-check` and `2>/dev/null`. Only override model when the user explicitly requests it, or when the task clearly fits the Spark speed-first criteria above.

| Use case | Command |
| --- | --- |
| Read-only analysis | `codex exec --skip-git-repo-check -o "$CODEX_OUT" "prompt" 2>/dev/null` |
| Speed-first interactive loop (Spark) | add `--model gpt-5.3-codex-spark` |
| Apply local edits | add `--profile edit` |
| Network / broad access | add `--profile full` |
| Resume session | `echo "prompt" \| codex exec --skip-git-repo-check -o "$CODEX_OUT" resume --last 2>/dev/null` |

Create a temp file for output before running: `CODEX_OUT=$(mktemp /tmp/codex-out.XXXXXX)`. Read the result from `$CODEX_OUT` after completion.

Run with Bash `run_in_background: true`. Monitor via `TaskOutput` with `block: true, timeout: 600000` — do NOT use `sleep` commands to poll. Read the `-o` file for the final answer.

## Resuming

Don't pass model/effort/sandbox flags when resuming — original session settings carry over. `-o` is safe to include.

## Conventions

This is a Go project. Use `go test`, `go build`, `go vet` in prompts. Codex workers don't receive CLAUDE.md or hooks, so project conventions must be explicit.

## Post-Execution Verification

**Always verify Codex output via `git diff --stat`** — never trust the worker's self-reported file list. Codex workers may make out-of-scope changes (deleting files, reverting unrelated code, removing test functions). The completion promise may list only intended files while omitting destructive side effects.

When other work exists in the same codebase, include explicit "DO NOT modify or delete" file lists in the prompt. The positive instruction ("only touch these files") is insufficient — Codex interprets "clean up" broadly.

Per [[principles/prove-it-works]]: trust artifacts, not self-reports.

## Worktree Usage

With `-C <worktree-dir>`, the worktree has its own git index. `.claude/` may symlink to `.agents/` — use `.agents/` paths for git operations.
