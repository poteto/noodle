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
		{Name: "skills", Short: "List resolved skills", Subcommands: []Command{
			{Name: "list", Short: "List all resolved skills"},
		}},
		{Name: "schema", Short: "Print generated schema docs for Noodle runtime contracts", Subcommands: []Command{
			{Name: "list", Short: "List available schema targets"},
		}},
		{Name: "worktree", Short: "Manage linked git worktrees", Subcommands: []Command{
			{Name: "create", Short: "Create a new linked worktree", Flags: []Flag{
				{Name: "from", Type: "string", Desc: "Branch or commit to base the new worktree on (default: HEAD)"},
			}},
			{Name: "exec", Short: "Run command inside worktree (CWD-safe)"},
			{Name: "merge", Short: "Merge a worktree branch into a target branch", Flags: []Flag{
				{Name: "into", Type: "string", Desc: "Target branch to merge into (default: integration branch)"},
			}},
			{Name: "cleanup", Short: "Remove a worktree without merging", Flags: []Flag{
				{Name: "force", Type: "bool", Desc: "Remove even when unmerged commits exist"},
			}},
			{Name: "list", Short: "List all worktrees with merge status"},
			{Name: "prune", Short: "Remove merged and patch-equivalent worktrees"},
			{Name: "hook", Short: "Run worktree session hook"},
		}},
		{Name: "event", Short: "Manage loop events", Subcommands: []Command{
			{Name: "emit", Short: "Emit an external event", Flags: []Flag{
				{Name: "payload", Type: "string", Desc: "Event payload as JSON"},
			}},
		}},
		{Name: "reset", Short: "Clear all runtime state"},
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
