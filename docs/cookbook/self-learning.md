# Self-Learning

This page describes the [brainmaxxing](https://github.com/poteto/brainmaxxing) skill pack, an optional addon that gives Noodle agents persistent memory across sessions. Brainmaxxing is not part of Noodle's core. The noodle loop works without it.

Brainmaxxing adds three skills that form a self-learning loop on top of the [brain](/concepts/brain) (an Obsidian-compatible directory of markdown files). Together they make the brain get better over time: reflect, meditate, and ruminate.

## The loop

Each skill operates at a different timescale:

- **reflect** runs after every agent session. It captures what just happened.
- **meditate** runs periodically, after several reflect cycles have accumulated. It distills and prunes.
- **ruminate** runs on its own schedule to mine past conversations for knowledge that was never captured.

Reflect captures. Meditate distills. Ruminate fills gaps.

## Reflect

*Part of [brainmaxxing](https://github.com/poteto/brainmaxxing).*

```yaml
---
name: reflect
description: Reflect on the conversation and update the brain.
schedule: "After an agent session completes"
---
```

After an agent finishes work (implementing a feature, fixing a bug, running a deploy) the reflect skill looks at what happened and writes it down. It creates or updates brain files with:

- What worked and what didn't
- Patterns the agent discovered while coding
- Corrections to existing brain content that turned out to be wrong
- New project-specific knowledge (API quirks, testing gotchas, deployment steps)

Reflect is fast and frequent. It runs after every session, so it captures information while the context is fresh. The notes it writes might be rough or redundant. That's fine. Meditate cleans them up later.

## Meditate

*Part of [brainmaxxing](https://github.com/poteto/brainmaxxing).*

```yaml
---
name: meditate
description: Audit and evolve the brain vault. Prune outdated content, discover cross-cutting principles.
schedule: "Periodically after several reflect cycles have accumulated"
---
```

Meditate reads the whole brain and looks for problems: outdated notes that no longer match the codebase, duplicate information scattered across files, patterns that appear in multiple places but haven't been named as principles.

It does three things:

1. **Prune**: delete notes that are stale or wrong. A reflection from two weeks ago about a bug that's since been fixed is clutter.
2. **Consolidate**: merge related notes into single, well-structured files. Five different reflections about the same API pattern should become one reference document.
3. **Extract principles**: when the same lesson appears in multiple reflections, promote it to `brain/principles.md` so future agents follow it from the start.

Meditate runs less often than reflect. It needs enough raw material to work with. A single reflection isn't worth auditing, but ten of them probably contain patterns worth distilling.

## Ruminate

*Part of [brainmaxxing](https://github.com/poteto/brainmaxxing).*

Ruminate is the scavenger. It goes back through past conversations and session logs looking for knowledge that was never captured by reflect. Maybe a cook solved a tricky problem but the reflection was too brief. Maybe a conversation contained a design decision that nobody wrote down.

Ruminate fills these gaps. It reads old transcripts, extracts the useful parts, and writes brain files for them. Over time, this means less knowledge is lost between sessions.

## What this looks like in practice

Session 1: An agent implements a new API endpoint. Reflect runs and notes that the test database needs a specific seed file.

Session 2: Another agent implements a different endpoint. Reflect notes the same database seed requirement.

Session 3: Meditate runs. It sees both reflections mention the seed file, consolidates them into a single `brain/testing/database-setup.md` file, and adds "Always run the seed file before integration tests" to `brain/principles.md`.

Session 4: The next agent reads `brain/principles.md` before starting work. It knows about the seed file requirement without having to discover it the hard way.

Session 5: Ruminate reviews older session logs and finds a conversation where the user explained why the seed file exists and what happens if you skip it. It adds this context to `brain/testing/database-setup.md`.

Each session leaves the brain a little better than it found it. The agent gets smarter not by having a larger model, but by having better notes.

## Setup

Install the brainmaxxing skill pack from [github.com/poteto/brainmaxxing](https://github.com/poteto/brainmaxxing). The repo includes installation instructions and the three skill definitions (reflect, meditate, ruminate) ready to drop into your `.agents/skills/` directory.
