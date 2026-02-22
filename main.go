package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/poteto/noodle/config"
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

	loadedConfig, validation, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return err
	}
	app := &App{
		Config:     loadedConfig,
		Validation: validation,
	}
	if err := reportConfigDiagnostics(os.Stderr, commandName, validation); err != nil {
		return err
	}

	return command.Run(ctx, app, catalog, args[1:])
}

func reportConfigDiagnostics(w io.Writer, commandName string, validation config.ValidationResult) error {
	if len(validation.Diagnostics) == 0 {
		return nil
	}

	for _, diagnostic := range validation.Diagnostics {
		line := fmt.Sprintf(
			"config %s: %s: %s",
			strings.ToLower(string(diagnostic.Severity)),
			diagnostic.FieldPath,
			diagnostic.Message,
		)
		if diagnostic.Fix != "" {
			line += " Fix: " + diagnostic.Fix
		}
		fmt.Fprintln(w, line)
	}

	if commandName == "start" && len(validation.Fatals()) > 0 {
		return fmt.Errorf("fatal config diagnostics prevent start")
	}

	return nil
}
