package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
)

func runMiseCommand(ctx context.Context, app *App, _ []Command, args []string) error {
	flags := flag.NewFlagSet("mise", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}

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
