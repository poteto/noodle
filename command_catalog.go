package main

import (
	"context"

	"github.com/poteto/noodle/config"
)

type App struct {
	Config     config.Config
	Validation config.ValidationResult
}

// Command is the unified CLI command definition used across all phases.
type Command struct {
	Name        string
	Description string
	Category    string
	Run         func(ctx context.Context, app *App, catalog []Command, args []string) error
}

func CommandCatalog() []Command {
	return []Command{
		{
			Name:        "commands",
			Description: "List available commands",
			Category:    "core",
			Run:         runCommandsCommand,
		},
		{
			Name:        "skills",
			Description: "List resolved skills",
			Category:    "core",
			Run:         runSkillsCommand,
		},
		{
			Name:        "stamp",
			Description: "Stamp NDJSON logs and emit canonical sidecar events",
			Category:    "internal",
			Run:         runStampCommand,
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
