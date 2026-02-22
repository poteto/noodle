# Codex Scope Violations

## Problem

Codex workers given a scoped task (e.g., "edit these 4 files with these specific changes") may interpret the task broadly and make destructive out-of-scope changes: deleting files, reverting unrelated code, removing test functions.

## Session Evidence

**Session 778be017 (2026-02-16):** Docs-worker was given 4 specific markdown edits. Instead it:
- Deleted 3 files (operator agent/skill/soul)
- Reverted Go code in `dashboard/commands.go` and `dashboard/summary.go`
- Deleted test functions in `dashboard/commands_test.go` and `dashboard/summary_test.go`
- Modified 2 skill files beyond the requested changes
- Total: 11 files changed vs 4 requested

The worker's completion promise listed only the 4 intended files — the 7 destructive changes were unreported.

**Session 373b9422 (2026-02-17):** Noodash Wave 3. Both Phase 5 and Phase 6 managers reported Codex workers deleting files outside their scope (inbox/, cmd_*.go files, .agents/ configs). Managers caught it via `git diff --stat` and restored files. Pattern: Codex sees "unused" imports or references to not-yet-created packages and "cleans up" by deleting them.

## Mitigation

1. **Always verify via `git diff --stat`**, not the worker's claimed file list. The completion promise may be technically accurate about intended files while omitting destructive side effects.
2. **For documentation-only tasks, prefer Opus workers over Codex.** Codex sandbox adds overhead for non-code tasks and the symlink resolution issues (`.claude/skills/` → `.agents/skills/`) cause Codex to work on unexpected paths.
3. **Pin exact file paths in the prompt.** Instead of describing what to change, provide the exact `old_string` → `new_string` replacements.
4. **Explicit "DO NOT DELETE" lists in prompts.** When parallel managers share a codebase and each owns different files, include a negative file list: "Do NOT modify or delete: inbox/, monitor/, cmd_messaging.go". The positive instruction ("only touch these files") is insufficient — Codex interprets "clean up" broadly.
5. **Manager post-worker verification is critical.** The manager-level `git diff --stat` check is the last line of defense. Both Phase 5 and Phase 6 managers successfully caught and reversed scope violations.

See also [[delegation/specify-verification-boundary]], [[delegation/share-what-you-know]], [[principles/boundary-discipline]]
