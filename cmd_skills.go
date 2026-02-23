package main

import (
	"fmt"
	"os"

	"github.com/poteto/noodle/skill"
	"github.com/spf13/cobra"
)

func newSkillsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "List resolved skills",
	}
	cmd.AddCommand(newSkillsListCmd(app))
	return cmd
}

func newSkillsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all resolved skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSkillsList(app)
		},
	}
}

func runSkillsList(app *App) error {
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
