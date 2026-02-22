package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

func runFixturesCommand(_ context.Context, _ *App, _ []Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("fixtures subcommand is required")
	}

	subcommand := strings.TrimSpace(args[0])
	switch subcommand {
	case "sync":
		return runFixturesSyncSubcommand(args[1:], false)
	case "check":
		return runFixturesSyncSubcommand(args[1:], true)
	default:
		return fmt.Errorf("unknown fixtures subcommand %q", subcommand)
	}
}

func runFixturesSyncSubcommand(args []string, checkOnly bool) error {
	name := "fixtures sync"
	if checkOnly {
		name = "fixtures check"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "Repository root to scan for fixtures")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%s does not accept positional arguments", name)
	}

	paths, err := fixturedir.SyncExpectedMarkdown(strings.TrimSpace(*root), checkOnly)
	if err != nil {
		for _, path := range paths {
			fmt.Fprintln(os.Stderr, path)
		}
		if checkOnly {
			return fmt.Errorf("fixture expected.md files are out of date; run `noodle fixtures sync`: %w", err)
		}
		return err
	}

	if checkOnly {
		fmt.Fprintln(os.Stdout, "fixtures check: up to date")
		return nil
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stdout, "fixtures sync: no changes")
		return nil
	}
	for _, path := range paths {
		fmt.Fprintln(os.Stdout, path)
	}
	fmt.Fprintf(os.Stdout, "fixtures sync: updated %d file(s)\n", len(paths))
	return nil
}
