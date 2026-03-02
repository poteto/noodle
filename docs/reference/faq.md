# FAQ

## Why not just use an agent directly?

You can. A single agent session with a good prompt gets work done. But you decide what to work on, you review the output, you context-switch between tasks, you run one thing at a time.

Noodle automates that loop. A scheduling agent reads your backlog and picks work. Multiple agents run in parallel, each in an isolated worktree. Completed work merges back and the scheduler re-evaluates.

## Where does the backlog come from?

Anywhere. A `todos.md` file works out of the box. [Adapters](/concepts/adapters) let Noodle pull from GitHub Issues, Linear, Jira, or whatever you already use.

## Can agents run in the cloud?

Yes. Noodle can run agents as local child processes or on cloud VMs. The scheduler picks the runtime for each task. See [Runtimes](/concepts/runtimes).

## Do agents need persistent memory?

No. Noodle works without it. But if you want agents that learn across sessions, [brainmaxxing](https://github.com/poteto/brainmaxxing) adds `reflect`, `meditate`, and `ruminate` skills that capture learnings and feed them back into future sessions. It's a separate install.

## Who is this for?

Developers who already use AI coding agents and want to scale up. You've used Claude Code, Codex, or Cursor. You've seen what one session can do. Now you want multiple sessions in parallel, automatic scheduling, and agents that build on each other's work.
