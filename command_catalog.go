package main

import "context"

// Command is the unified CLI command definition used across all phases.
type Command struct {
	Name        string
	Description string
	Category    string
	Run         func(ctx context.Context, catalog []Command, args []string) error
}

func CommandCatalog() []Command {
	return []Command{
		{
			Name:        "commands",
			Description: "List available commands",
			Category:    "core",
			Run:         runCommandsCommand,
		},
	}
}

func FindCommand(catalog []Command, name string) (Command, bool) {
	for _, command := range catalog {
		if command.Name == name {
			return command, true
		}
	}
	return Command{}, false
}
