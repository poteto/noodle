package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/worktree"
	"github.com/spf13/cobra"
)

type worktreeCommandApp interface {
	Create(name string) error
	Exec(name string, args []string) error
	Merge(name string) error
	Cleanup(name string, force bool) error
	List() error
	Prune() error
}

var newWorktreeCommandApp = func() (worktreeCommandApp, error) {
	app, err := worktree.NewApp()
	if err != nil {
		return nil, err
	}
	app.CmdPrefix = "noodle worktree"
	return app, nil
}

var runWorktreeHook = worktree.RunHook

func newWorktreeCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage linked git worktrees",
	}
	cmd.AddCommand(
		newWorktreeHookCmd(),
		newWorktreeCreateCmd(),
		newWorktreeExecCmd(),
		newWorktreeMergeCmd(),
		newWorktreeCleanupCmd(),
		newWorktreeListCmd(),
		newWorktreePruneCmd(),
	)
	return cmd
}

func newWorktreeHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Run worktree session hook",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWorktreeHook(os.Stdin, os.Stdout)
		},
	}
}

func newWorktreeCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new linked worktree",
		Args:  exactTrimmedArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.Create(strings.TrimSpace(args[0]))
		},
	}
}

func newWorktreeExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "exec <name> <command...>",
		Short:              "Run command inside worktree (CWD-safe)",
		Args:               cobra.MinimumNArgs(2),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.Exec(strings.TrimSpace(args[0]), args[1:])
		},
	}
}

func newWorktreeMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <name>",
		Short: "Merge a worktree branch back to main",
		Args:  exactTrimmedArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.Merge(strings.TrimSpace(args[0]))
		},
	}
}

func newWorktreeCleanupCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "cleanup <name>",
		Short: "Remove a worktree without merging",
		Args:  exactTrimmedArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.Cleanup(strings.TrimSpace(args[0]), force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Remove even when unmerged commits exist")
	return cmd
}

func newWorktreeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with merge status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.List()
		},
	}
}

func newWorktreePruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove merged and patch-equivalent worktrees",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wApp, err := newWorktreeCommandApp()
			if err != nil {
				return err
			}
			return wApp.Prune()
		},
	}
}

// exactTrimmedArgs returns a cobra arg validator that requires exactly n
// non-empty (after trimming) arguments.
func exactTrimmedArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		for _, arg := range args {
			if strings.TrimSpace(arg) == "" {
				return fmt.Errorf("argument must not be empty")
			}
		}
		return nil
	}
}
