package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/worktree"
)

type worktreeCommandApp interface {
	Create(name string) error
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

func runWorktreeCommand(_ context.Context, _ *App, _ []Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("worktree subcommand is required")
	}

	subcommand := strings.TrimSpace(args[0])
	switch subcommand {
	case "hook":
		if len(args) != 1 {
			return fmt.Errorf("worktree hook does not accept arguments")
		}
		return runWorktreeHook(os.Stdin, os.Stdout)
	case "create", "merge", "cleanup", "list", "prune":
		// handled below
	default:
		return fmt.Errorf("unknown worktree subcommand %q", subcommand)
	}

	wApp, err := newWorktreeCommandApp()
	if err != nil {
		return err
	}

	switch subcommand {
	case "create":
		name, err := requiredWorktreeName("create", args[1:])
		if err != nil {
			return err
		}
		return wApp.Create(name)
	case "merge":
		name, err := requiredWorktreeName("merge", args[1:])
		if err != nil {
			return err
		}
		return wApp.Merge(name)
	case "cleanup":
		return runWorktreeCleanupSubcommand(wApp, args[1:])
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("worktree list does not accept arguments")
		}
		return wApp.List()
	case "prune":
		if len(args) != 1 {
			return fmt.Errorf("worktree prune does not accept arguments")
		}
		return wApp.Prune()
	default:
		return fmt.Errorf("unknown worktree subcommand %q", subcommand)
	}
}

func requiredWorktreeName(subcommand string, args []string) (string, error) {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("worktree %s requires a name", subcommand)
	}
	return strings.TrimSpace(args[0]), nil
}

func runWorktreeCleanupSubcommand(wApp worktreeCommandApp, args []string) error {
	flags := flag.NewFlagSet("worktree cleanup", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	force := flags.Bool("force", false, "Remove even when unmerged commits exist")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("worktree cleanup requires a name")
	}
	return wApp.Cleanup(strings.TrimSpace(flags.Arg(0)), *force)
}
