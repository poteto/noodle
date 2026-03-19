# Domain Skills Routing

## Match Known Skills

Check installed skills (`.claude/skills/`) for any that match the plan's domain. Common matches:

| Domain | Skill | When |
|--------|-------|------|
| Codex delegation | `codex` | Tasks delegated to Codex workers |

**Invoke matched skills now** — read their output and incorporate domain guidance into the plan.

## Discover Missing Skills

If the plan touches a domain **not covered** by the table above, use the `find-skills` skill to search for a relevant skill. If one is found, install it (without `-g` — project-local only) and incorporate its guidance into the plan. Note what was installed so the user can see it. After the plan is written, delete any one-off skills that won't be needed again.
