package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/mise"
	"github.com/spf13/cobra"
)

func newMiseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "mise",
		Short: cmdmeta.Short("mise"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMise(cmd.Context(), app)
		},
	}
}

func runMise(ctx context.Context, app *App) error {
	cwd, err := app.ProjectDir()
	if err != nil {
		return err
	}

	builder := mise.NewBuilder(cwd, app.Config)
	resolver := app.SkillResolver()
	if taskTypeSkills, discoverErr := resolver.DiscoverTaskTypes(); discoverErr == nil {
		summaries := make([]mise.TaskTypeSummary, len(taskTypeSkills))
		for i, s := range taskTypeSkills {
			summaries[i] = mise.TaskTypeSummary{
				Key:      s.Name,
				CanMerge: s.Frontmatter.Noodle.Permissions.CanMerge(),
				Schedule: s.Frontmatter.Noodle.Schedule,
			}
		}
		builder.TaskTypes = summaries
	}
	brief, warnings, err := builder.Build(ctx, mise.ActiveSummary{}, nil)
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
