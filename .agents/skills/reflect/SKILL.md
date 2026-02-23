---
name: reflect
description: Capture learnings from mistakes and corrections in cook sessions. Persists to brain/, skill files, or structural enforcement.
noodle:
  blocking: false
  schedule: "After a cook session completes"
---

# Reflect

Capture mistakes made and corrections received. Route each learning to the destination with the highest leverage.

## Process

1. **Read `brain/index.md`** — understand existing notes. Skip duplicates.
2. **Scan the session** for mistakes and corrections only:
   - Errors you made and how you were corrected
   - Wrong assumptions that led to wasted work
   - Patterns you missed that caused bugs or rework
3. **Structural enforcement check** — for each learning, ask: can this become a lint rule, script, CI check, or skill instruction? If yes, encode it there. If no, write a brain note.
4. **Route each learning:**

| Destination | When |
|---|---|
| Lint rule / script / CI check | Learning can be mechanically enforced |
| `.agents/skills/<skill>/SKILL.md` | Learning is specific to how a skill operates |
| `brain/` note | Learning requires judgment, not mechanical enforcement |
| `brain/todos.md` | Follow-up work needed that can't be done now |

5. **Update `brain/index.md`** if brain files were added or removed.

## Brain note conventions

- One topic per file. File name = topic slug.
- Group in directories with index files using `[[wikilinks]]`.
- No inlined content in index files.

## Summary

After routing, output:

```
## Reflect Summary
- Brain: [files created/updated, one-line each]
- Skills: [skill files modified, one-line each]
- Structural: [rules/scripts/checks added]
- Todos: [follow-up items filed]
```
