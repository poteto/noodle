package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
	"github.com/spf13/cobra"
)

func newMiseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "mise",
		Short: "Build and print the current mise brief",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMise(cmd.Context(), app)
		},
	}
}

func runMise(ctx context.Context, app *App) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	builder := mise.NewBuilder(cwd, app.Config)
	resolver := skill.Resolver{SearchPaths: app.Config.Skills.Paths}
	if taskTypeSkills, discoverErr := resolver.DiscoverTaskTypes(); discoverErr == nil {
		summaries := make([]mise.TaskTypeSummary, len(taskTypeSkills))
		for i, s := range taskTypeSkills {
			summaries[i] = mise.TaskTypeSummary{
				Key:      s.Name,
				Blocking: s.Frontmatter.Noodle.Blocking,
				Schedule: s.Frontmatter.Noodle.Schedule,
			}
		}
		builder.TaskTypes = summaries
	}
	brief, warnings, err := builder.Build(ctx)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(brief); err != nil {
		return fmt.Errorf("encode mise output: %w", err)
	}
	return nil
}
