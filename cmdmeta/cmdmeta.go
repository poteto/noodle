// Package cmdmeta defines CLI command metadata shared between root command
// wiring and the noodle skill generator. This is the single source of truth
// for command names and descriptions.
package cmdmeta

// Command describes a CLI command or subcommand.
type Command struct {
	Name        string
	Short       string
	Subcommands []Command
}

// Commands returns the full command tree metadata.
func Commands() []Command {
	return []Command{
		{Name: "start", Short: "Run the scheduling loop"},
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
			{Name: "cleanup", Short: "Remove a worktree without merging"},
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
		{Name: "stamp", Short: "Stamp NDJSON logs and emit canonical sidecar events"},
		{Name: "dispatch", Short: "Dispatch a cook session in tmux"},
		{Name: "mise", Short: "Build and print the current mise brief"},
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
