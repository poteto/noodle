# Philosophy

## The brain

Noodle projects have a brain — an Obsidian vault committed to the repo at `brain/`. It's not a chat log. It's structured, topic-per-file memory that agents read before they start working and write to after they learn something.

Principles, patterns, past mistakes, codebase quirks, architectural decisions. The brain holds the kind of knowledge that usually lives in someone's head and gets lost when they leave. Here it's written down and version-controlled, so every agent session starts with the project's accumulated understanding.

The brain is committed to git because it should evolve with the project. When the architecture changes, the brain updates. When a principle proves wrong, it gets deleted. It's living documentation maintained by agents, not a dusty wiki nobody reads.

## Self-learning

Three skills form the learning loop:

**Reflect** runs after significant sessions. The agent looks at what happened — what worked, what broke, what was surprising — and writes it to the brain. Maybe a test pattern keeps failing, or a certain API has a quirk that wastes time. The next agent reads that note and avoids the same trap.

**Meditate** runs periodically. It audits the brain for stale content, discovers cross-cutting principles from individual notes, and prunes what's no longer true. Over time, isolated observations condense into principles that shape every future decision.

**Ruminate** mines past conversations for knowledge that was never captured. Corrections, workarounds, patterns that someone figured out but didn't write down. It cross-references with the existing brain to fill gaps.

The result is a system that gets smarter without anyone curating it. Each session leaves the brain a little better than it found it.

## Why agent-pilled

Humans are the scarcest resource. Every step that blocks on human attention is waste. If an agent can make a reasonable decision, it should make it, proceed, and let the human course-correct after the fact.

This doesn't mean agents run wild. Noodle has a quality gate — every piece of work gets reviewed before it merges. But the review happens after the work is done, not before it starts. The human's job is strategy and judgment: choosing what to build, reviewing what was built, steering when things drift. Not execution.

Most teams do it backwards. They have agents wait for approval to start, then trust the output blindly. Noodle does the opposite: let agents start immediately, then verify everything they produce.

## Autonomy as a dial

Trust isn't binary. Noodle has three modes:

- **Full** — agents merge their own work. No human in the loop. For trusted projects with good test coverage.
- **Review** — agents work, the quality gate reviews, and approved work merges automatically. The default.
- **Approve** — same as review, but humans confirm before merging. For high-stakes changes.

The dial moves based on trust, not capability. A mature project with comprehensive tests can run in full autonomy. A new project starts in review mode. Critical infrastructure stays in approve mode. You decide where each project sits.

## Everything is a file

`.noodle/queue.json` is the work queue. `.noodle/mise.json` is the gathered project state. Session logs are NDJSON. Control commands are NDJSON. Verdicts are files. Config is TOML.

No databases. No hidden state. No proprietary formats. Every piece of Noodle's runtime state is a file you can read with `cat`, diff with `git`, or process with `jq`. When something goes wrong, you can see exactly what happened by reading the files.

This also means composition is free. Agents pass state by writing files. The scheduling agent reads the queue and writes a new one. A cook reads its assignment and writes its output. External tools can hook in by reading and writing the same files. There's no integration layer because there doesn't need to be one.

## Skills as the single extension point

There are no plugins, no hooks API, no configuration DSL. A skill is a `SKILL.md` file — markdown that teaches an agent how to do something. Users extend Noodle by writing skills.

Skills are composable. A scheduling skill reads the mise and writes the queue. An execution skill picks up a task and does the work. A review skill checks the output. You can swap any of them with your own version, and the system keeps working because the contract between them is just files on disk.

This means Noodle stays small. The Go binary handles mechanics — spawning sessions, monitoring health, gathering state. Everything that requires judgment or domain knowledge lives in skills. The binary doesn't know about your project, your workflow, or your preferences. The skills do.
