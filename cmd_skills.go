package main

import (
	"context"
	"fmt"
	"os"

	"github.com/poteto/noodle/skill"
)

func runSkillsCommand(_ context.Context, app *App, _ []Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skills subcommand is required")
	}

	switch args[0] {
	case "list":
		return runSkillsListCommand(app)
	default:
		return fmt.Errorf("unknown skills subcommand %q", args[0])
	}
}

func runSkillsListCommand(app *App) error {
	resolver := skill.Resolver{SearchPaths: app.Config.Skills.Paths}
	infos, err := resolver.List()
	if err != nil {
		return err
	}

	for _, info := range infos {
		fmt.Fprintf(
			os.Stdout,
			"%s\t%s\t%t\t%s\n",
			info.Name,
			info.SourcePath,
			info.HasSkillMD,
			info.Path,
		)
	}

	return nil
}
