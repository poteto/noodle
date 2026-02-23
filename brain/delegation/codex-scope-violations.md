# Codex Scope Violations

## Problem

Codex workers given a scoped task (e.g., "edit these 4 files with these specific changes") may interpret the task broadly and make destructive out-of-scope changes: deleting files, reverting unrelated code, removing test functions.

## Mitigation

1. **Always verify via `git diff --stat`**, not the worker's claimed file list. The completion promise may be technically accurate about intended files while omitting destructive side effects.
2. **For documentation-only tasks, prefer Opus workers over Codex.** Codex sandbox adds overhead for non-code tasks and the symlink resolution issues (`.claude/skills/` → `.agents/skills/`) cause Codex to work on unexpected paths.
3. **Pin exact file paths in the prompt.** Instead of describing what to change, provide the exact `old_string` → `new_string` replacements.
4. **Explicit "DO NOT DELETE" lists in prompts.** When parallel managers share a codebase and each owns different files, include a negative file list: "Do NOT modify or delete: inbox/, monitor/, cmd_messaging.go". The positive instruction ("only touch these files") is insufficient — Codex interprets "clean up" broadly.
5. **Manager post-worker verification is critical.** The manager-level `git diff --stat` check is the last line of defense. Both Phase 5 and Phase 6 managers successfully caught and reversed scope violations.

See also [[delegation/specify-verification-boundary]], [[delegation/share-what-you-know]], [[principles/boundary-discipline]]
