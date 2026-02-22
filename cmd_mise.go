package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/poteto/noodle/mise"
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
