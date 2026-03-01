# Philosophy

## The brain

Noodle projects have a brain, an Obsidian vault committed to the repo at `brain/`. Not a chat log. Structured, topic-per-file memory that agents read before they start and write to after they learn something.

Principles, patterns, past mistakes, codebase quirks, architectural decisions. The stuff that usually lives in someone's head and gets lost when they leave. Here it's written down and version-controlled, so every agent starts with what the project actually knows.

The brain is in git because it should evolve with the project. Architecture changes, brain updates. A principle proves wrong, it gets deleted. Agents maintain it, not humans.

## Self-learning

Three skills form the learning loop:

**Reflect** runs after significant sessions. The agent looks at what happened, what worked, what broke, what was surprising, and writes it to the brain. Maybe a test pattern keeps failing, or a certain API has a quirk that wastes time. The next agent reads that note and avoids the same trap.

**Meditate** runs periodically. It audits the brain for stale content, discovers cross-cutting principles from individual notes, and prunes what's no longer true. Over time, isolated observations condense into principles that shape every future decision.

**Ruminate** mines past conversations for knowledge that was never captured. Corrections, workarounds, patterns that someone figured out but didn't write down. It cross-references with the existing brain to fill gaps.

Nobody curates this. Each session leaves the brain a little better than it found it.

## Why agent-pilled

Humans are the scarcest resource. Every step that blocks on human attention is waste. If an agent can make a reasonable decision, it should make it, proceed, and let the human course-correct after the fact.

This doesn't mean agents run wild. There's a quality gate. Work gets reviewed before it merges. But the review happens after the work is done, not before it starts. Your job is strategy and judgment. Not execution.

Most teams do it backwards. Agents wait for approval to start, then the output gets trusted blindly. Noodle flips that: let agents start immediately, then verify what they produce.

## Autonomy as a dial

Trust isn't binary. Noodle has three modes:

- **Full.** Agents merge their own work. No human in the loop. For trusted projects with good test coverage.
- **Review.** Agents work, the quality gate reviews, and approved work merges automatically. The default.
- **Approve.** Same as review, but humans confirm before merging. For high-stakes changes.

The dial moves based on trust, not capability. A mature project with good tests can run full autonomy. A new project starts in review. Critical infrastructure stays in approve. You decide.

## Everything is a file

`.noodle/orders.json` is the work orders. `.noodle/mise.json` is the gathered project state. Session logs are NDJSON. Control commands are NDJSON. Verdicts are files. Config is TOML.

No databases. No hidden state. No proprietary formats. All of Noodle's runtime state is a file you can `cat`, `git diff`, or `jq`. When something breaks, you read the files and see what happened.

Composition falls out of this for free. Agents pass state by writing files. The scheduler reads the queue and writes a new one. A cook reads its task and writes its output. External tools hook in the same way. Read a file, write a file. No integration layer needed.

## Skills as the single extension point

There are no plugins, no hooks API, no configuration DSL. A skill is a `SKILL.md` file, markdown that teaches an agent how to do something. Users extend Noodle by writing skills.

They compose naturally. A scheduling skill reads the mise and writes orders. An execution skill picks up a task and does the work. Swap any of them with your own version. The contract between skills is just files on disk.

This keeps Noodle small. The Go binary handles mechanics: spawning sessions, monitoring health, gathering state. Judgment and domain knowledge live in skills. The binary doesn't know your project. The skills do.
