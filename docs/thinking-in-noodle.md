# Thinking in Noodle

The [Introduction](/introduction) and [Getting Started](/getting-started) cover what Noodle does. This page is about how to think when using it.

## Start with what you repeat

Before writing any skills, watch what you actually do. Over a week of working with agents, you'll notice patterns:

- You paste the same debugging instructions every time something breaks
- You review every PR the same way
- You remind agents about your commit conventions
- You check the backlog every morning and decide what to work on next

Those are your skills. Write down what you already do. Ideally, everything you do could be captured in skills.

## Write it down before you automate

When you spot a pattern, write a general skill first. No `schedule:` field. Just the knowledge.

```yaml
---
name: review
description: >
  Review code changes for correctness and style.
---

Walk the diff in this order:
1. Architecture — does the change belong here?
2. Correctness — does it do what it claims?
3. Tests — are the important paths covered?
4. Performance — anything obviously expensive?

Number each issue. For each one, suggest two options
with tradeoffs. Don't nitpick formatting.
```

Use it manually for a while. You'll find out what's missing. Maybe you forgot to mention how to handle failing tests. Maybe the ordering doesn't matter as much as you thought. Edit the skill as you learn.

Separate the "what do I know" step from the "when should it run" step. Mixing them makes both harder.

## One skill, one job

A skill that reviews code AND deploys AND notifies Slack is hard to schedule. When should it run? After every commit? Only on main? The `schedule:` field becomes a paragraph of conditions and exceptions.

Split it. `review` runs after each session. `deploy` runs after review passes on main. `notify` runs after deploy. Each one has a clear trigger.

The test: can you describe when this skill should run in one sentence? If you need "and" more than once, it's probably two skills.

## Describe when, not how to trigger

If you're coming from traditional automation, this is the mental shift. You're not registering event handlers. You're describing conditions.

```yaml
# not this
schedule: "0 9 * * MON-FRI"

# this
schedule: >
  When the backlog has new items that haven't
  been planned yet
```

The scheduler reads these as prose and uses judgment. It checks the current project state and decides if the conditions fit. So you can write things cron can't express: "when there's a failed task that hasn't been retried" or "after a batch of related items lands in the backlog."

Start vague, tighten as you see what the scheduler does.

## Let the scheduler compose

With traditional tools, you'd wire skills together: review calls deploy, deploy calls notify. In Noodle, each skill describes when it should run, and the scheduler figures out the order.

This feels weird at first. You might want to write `after: review` in your deploy skill. But the scheduler already knows that review runs after execute. It reads all the `schedule:` fields together and sequences them.

This means you can add a new skill without touching existing ones. Write a `security-scan` skill with `schedule: After review passes, before deploy`, and it slots into the pipeline.

## Start with two skills

You don't need ten skills on day one.

- **schedule** reads the backlog and picks what to work on
- **execute** picks up a task and does the work

That's a working noodle loop. Run it. Watch what happens. You'll quickly feel what's missing. Agents commit with bad messages? Write a `commit` skill. You're reviewing every PR yourself? Write a `review` skill and schedule it. Agents keep making the same mistakes? Add a `reflect` skill.

Your skill library grows from what you notice, not what you plan.

## Writing good skill bodies

The body of a skill is a prompt. You're teaching an agent how to do a job.

Be specific about process, loose about implementation. "Walk the diff in this order: architecture, correctness, tests" is good. "Use `git diff HEAD~1` and parse the output" is too prescriptive. The agent knows how to use git. Tell it what to look for.

Include your reasoning. Not just "fix the root cause, not the symptom" but why: "patching symptoms creates a second bug when the root cause resurfaces." Agents make better calls when they understand the reason behind a rule.

Write for a smart coworker on their first day. They know how to code. They don't know your conventions, your preferences, or the gotchas you've learned the hard way.

If you use Claude Code, install Anthropic's [skill-creator](https://github.com/anthropics/skills/tree/main/skills/skill-creator). It walks you through writing a well-structured skill and handles the frontmatter for you. We recommend using it for every new skill.

```sh
npx skills add https://github.com/anthropics/skills --skill skill-creator
```

## What to read next

- [Getting Started](/getting-started) — install Noodle and get a loop running
- [Skills](/concepts/skills) — scheduled vs general skills, discovery, composition
- [Scheduling](/concepts/scheduling) — how the noodle loop works under the hood
