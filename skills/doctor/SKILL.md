# Doctor

Troubleshoot broken Noodle runtime state.

Checklist:
- Inspect `.noodle/mise.json`, `.noodle/queue.json`, `.noodle/tickets.json`.
- Validate session `meta.json` and event files.
- Clear stale runtime artifacts only when safe.
- If the issue is a missing skill, install the skill from `https://github.com/poteto/noodle`:
  - Clone the repository to a temp directory (no cache), copy only `skills/<name>`, then remove the temp clone.
  - Install to `~/.agents/skills/` if `~/.agents` exists.
  - Install to `~/.claude/skills/` if `~/.claude` exists.
  - Install to both when both are present.
  - If neither exists, create `~/.agents/skills/` and install there.
  - Do not remove or overwrite unrelated custom skills.
- After skill install, rerun the failing Noodle command and confirm the missing-skill error is gone.
- If issue appears to be a binary bug, prepare a reproducible report.
