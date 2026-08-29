package dispatcher

const noodleWorktreePreamble = `# Noodle Context

You are running inside a Noodle cook session — an autonomous coding agent managed by the Noodle framework.

## Available Files

Your working directory is a git worktree (an isolated checkout). It contains committed project files:

- ` + "`" + `todos.md` + "`" + ` — Project backlog items

The ` + "`" + `.noodle/` + "`" + ` directory (mise.json, orders.json, tickets.json) is in the main checkout, not in your worktree. Your task context is provided in the prompt — do not try to read .noodle state files.

## Conventions

- Work in your assigned worktree — do not modify the primary checkout
- Commit with conventional commit messages
- Run verification before finishing (tests, lint, build)
- Write learnings to brain/ files when you discover something notable
`

const noodlePrimaryCheckoutPreamble = `# Noodle Context

You are running inside a Noodle cook session — an autonomous coding agent managed by the Noodle framework.

## Working Directory

Your working directory is the primary checkout (not a linked worktree). The ` + "`" + `.noodle/` + "`" + ` directory (mise.json, orders.json, tickets.json) lives here. Your task context and any required paths are provided in the prompt.

## Conventions

- Commit with conventional commit messages
- Run verification before finishing (tests, lint, build)
- Write learnings to brain/ files when you discover something notable
`

// buildSessionPreamble returns the generic framework context injected into
// every session's system prompt. Its "Available Files" claims (isolated
// worktree, todos.md backlog file) only hold for sessions dispatched to a
// linked worktree — a session allowed to run on the primary checkout (e.g.
// schedule, skill bootstrap) gets the truthful primary-checkout variant
// instead, so it isn't told it's isolated when it isn't, or pointed at a
// default backlog file that may not exist for the project's adapter.
func buildSessionPreamble(allowPrimaryCheckout bool) string {
	if allowPrimaryCheckout {
		return noodlePrimaryCheckoutPreamble
	}
	return noodleWorktreePreamble
}
