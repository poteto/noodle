---
name: find-skills
description: >-
  Discover and install agent skills from the open ecosystem. Use IMMEDIATELY when the user
  asks "how do I do X", "find a skill for X", "is there a skill for X", "can you do X"
  for specialized capabilities, wants to extend agent functionality, mentions a domain
  they wish they had help with, or asks about tools/templates/workflows that might exist
  as installable skills.
---

# Find Skills

Search for and install skills from the open agent skills ecosystem.

## Commands

```bash
# Search for skills by keyword
pnpx skills find [query]

# Install a skill (always project-local, NEVER use -g)
pnpx skills add <owner/repo@skill> -y
```

**Never use `-g` (global flag)** — always install project-local.

**Browse available skills:** https://skills.sh/

## Workflow

1. Run `pnpx skills find [query]` with keywords matching the user's need
2. Present results with the skill name, what it does, and the install command
3. If the user wants it, install with `pnpx skills add <owner/repo@skill> -y`

## When No Results Found

1. Acknowledge no existing skill was found
2. Offer to help with the task directly using general capabilities
3. Mention `pnpx skills init my-skill-name` if they want to create their own
