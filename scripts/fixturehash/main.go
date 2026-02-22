package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("fixture hash mode is required (sync|check)")
	}
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	switch mode {
	case "sync":
		return runSync(false, args[1:])
	case "check":
		return runSync(true, args[1:])
	default:
		return fmt.Errorf("unknown fixture hash mode %q (expected sync|check)", mode)
	}
}

func runSync(checkOnly bool, args []string) error {
	name := "fixturehash sync"
	if checkOnly {
		name = "fixturehash check"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "Fixture root to scan")
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
		return err
	}

	if checkOnly {
		fmt.Fprintln(os.Stdout, "fixture hash check: up to date")
		return nil
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stdout, "fixture hash sync: no changes")
		return nil
	}
	for _, path := range paths {
		fmt.Fprintln(os.Stdout, path)
	}
	fmt.Fprintf(os.Stdout, "fixture hash sync: updated %d file(s)\n", len(paths))
	return nil
}
