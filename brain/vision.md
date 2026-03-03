# Noodle Vision

Agents read files. Agents write files. It's the one thing every agent is already good at.

Noodle is an agent orchestration framework where the only API is files. All system state lives in `.noodle/` — orders, sessions, status. A skill is a markdown file that tells an agent how to do something. Skills are to Noodle what components are to React.

The minimal autonomous system is two skills working together: **schedule** reads the backlog and decides what to work on next, **execute** picks up a task and does the work. From there you add more skills — quality, reflect, meditate — and the system gets smarter over time because agents rewrite the skills based on accumulated experience.

See [[architecture]] for the full technical walkthrough.
