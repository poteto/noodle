package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	catalog := CommandCatalog()
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}

	commandName := args[0]
	command, ok := FindCommand(catalog, commandName)
	if !ok {
		return fmt.Errorf("unknown command %q", commandName)
	}

	return command.Run(ctx, catalog, args[1:])
}
