// Package cmdmeta defines CLI command metadata shared between root command
// wiring and the noodle skill generator. This is the single source of truth
// for command names and descriptions.
package cmdmeta

// Flag describes a CLI flag.
type Flag struct {
	Name    string // long name (e.g. "once")
	Short   string // short name (e.g. "o"), empty if none
	Type    string // "bool", "string", "int", "float64", "[]string"
	Default string // default value as string, empty if zero
	Desc    string // description
}

// Command describes a CLI command or subcommand.
type Command struct {
	Name        string
	Short       string
	Flags       []Flag
	Subcommands []Command
}

// Commands returns the full command tree metadata.
func Commands() []Command {
	return []Command{
		{Name: "start", Short: "Run the scheduling loop", Flags: []Flag{
			{Name: "once", Type: "bool", Desc: "Run one scheduling cycle and exit"},
		}},
		{Name: "status", Short: "Show compact runtime status"},
		{Name: "debug", Short: "Dump canonical runtime debug state"},
		{Name: "skills", Short: "List resolved skills", Subcommands: []Command{
			{Name: "list", Short: "List all resolved skills"},
		}},
		{Name: "schema", Short: "Print generated schema docs for Noodle runtime contracts", Subcommands: []Command{
			{Name: "list", Short: "List available schema targets"},
		}},
		{Name: "worktree", Short: "Manage linked git worktrees", Subcommands: []Command{
			{Name: "create", Short: "Create a new linked worktree"},
			{Name: "exec", Short: "Run command inside worktree (CWD-safe)"},
			{Name: "merge", Short: "Merge a worktree branch back to main"},
			{Name: "cleanup", Short: "Remove a worktree without merging", Flags: []Flag{
				{Name: "force", Type: "bool", Desc: "Remove even when unmerged commits exist"},
			}},
			{Name: "list", Short: "List all worktrees with merge status"},
			{Name: "prune", Short: "Remove merged and patch-equivalent worktrees"},
			{Name: "hook", Short: "Run worktree session hook"},
		}},
		{Name: "plan", Short: "Manage plans (create, done, phase-add, list)", Subcommands: []Command{
			{Name: "create", Short: "Create a plan from a todo"},
			{Name: "activate", Short: "Mark a plan as active"},
			{Name: "done", Short: "Mark a plan as done"},
			{Name: "phase-add", Short: "Add a phase to a plan"},
			{Name: "list", Short: "List all plans"},
		}},
		{Name: "stamp", Short: "Stamp NDJSON logs and emit canonical sidecar events", Flags: []Flag{
			{Name: "output", Short: "o", Type: "string", Desc: "Output path for stamped NDJSON"},
			{Name: "events", Type: "string", Desc: "Output path for canonical sidecar events"},
		}},
		{Name: "dispatch", Short: "Dispatch a cook session in tmux", Flags: []Flag{
			{Name: "name", Type: "string", Default: "cook", Desc: "Session name"},
			{Name: "prompt", Type: "string", Desc: "Prompt text for the dispatched session"},
			{Name: "provider", Type: "string", Desc: "Provider (claude or codex)"},
			{Name: "model", Type: "string", Desc: "Model name"},
			{Name: "skill", Type: "string", Desc: "Skill name to inject"},
			{Name: "reasoning-level", Type: "string", Desc: "Reasoning level"},
			{Name: "worktree", Type: "string", Desc: "Linked worktree path"},
			{Name: "max-turns", Type: "int", Desc: "Max turns"},
			{Name: "budget-cap", Type: "float64", Desc: "Budget cap"},
			{Name: "env", Type: "[]string", Desc: "Extra env vars (KEY=VALUE)"},
		}},
		{Name: "mise", Short: "Build and print the current mise brief"},
		{Name: "event", Short: "Manage loop events", Subcommands: []Command{
			{Name: "emit", Short: "Emit an external event", Flags: []Flag{
				{Name: "payload", Type: "string", Desc: "Event payload as JSON"},
			}},
		}},
	}
}

// Short returns the Short description for a command by name path.
// For top-level: Short("start"). For sub: Short("plan", "create").
func Short(names ...string) string {
	cmds := Commands()
	for i, name := range names {
		for _, cmd := range cmds {
			if cmd.Name == name {
				if i == len(names)-1 {
					return cmd.Short
				}
				cmds = cmd.Subcommands
				break
			}
		}
	}
	return ""
}
