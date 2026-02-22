package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/poteto/noodle/tui"
)

func runTUI(runtimeDir string) error {
	model := tui.NewModel(tui.Options{
		RuntimeDir: runtimeDir,
	})
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func runTuiCommand(_ context.Context, _ *App, _ []Command, args []string) error {
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	runtimeDir := flags.String("runtime-dir", "", "Path to the .noodle runtime directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("tui does not accept arguments")
	}

	if strings.TrimSpace(*runtimeDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
		*runtimeDir = filepath.Join(cwd, ".noodle")
	}

	return runTUI(*runtimeDir)
}
