package dispatcher

const noodlePreamble = `# Noodle Context

You are running inside a Noodle cook session — an autonomous coding agent managed by the Noodle framework.

## State Files

- ` + "`" + `.noodle/mise.json` + "`" + ` — Current project state snapshot (backlog, plans, active cooks, resources, routing)
- ` + "`" + `.noodle/orders.json` + "`" + ` — Work orders
- ` + "`" + `.noodle/tickets.json` + "`" + ` — Active tickets and escalations
- ` + "`" + `.noodle/quality/` + "`" + ` — Post-cook quality verdict files
- ` + "`" + `brain/plans/` + "`" + ` — Implementation plans with phased execution
- ` + "`" + `brain/todos.md` + "`" + ` — Project backlog items

## Conventions

- Work in your assigned worktree — do not modify the primary checkout
- Commit with conventional commit messages
- Run verification before finishing (tests, lint, build)
- Write learnings to brain/ files when you discover something notable
`

func buildSessionPreamble() string { return noodlePreamble }
