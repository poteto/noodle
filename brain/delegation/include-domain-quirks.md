# Include Domain Quirks

When delegating work that involves a specific tool or system (Obsidian, Git, Cargo, etc.), include the non-obvious resolution rules and behavioral quirks in the prompt. These are the things that seem obvious once you know them but cost 3-5 turns to discover empirically.

Examples:
- Obsidian wikilinks resolve same-directory first, then fall back to full paths
- Git worktrees show `.claude/settings.local.json` as a typechange (symlink vs file)
- `cargo check` with new dependencies takes 2-5 minutes on first compile

The pattern: if you've seen a delegate spend turns discovering a quirk before, bake it into every future prompt that touches that domain. The cost of a longer prompt is one read; the cost of rediscovery is 3-5 turns of trial and error.

Domain quirks that recur across many delegations are candidates for structural encoding (lint rules, tool defaults, skill instructions) per [[principles/encode-lessons-in-structure]]. Prompt inclusion is the right delivery mechanism when the quirk is domain-specific and the structural fix would require unreasonable investment.

See also [[principles/cost-aware-delegation]], [[codebase/worktree-gotchas]]
