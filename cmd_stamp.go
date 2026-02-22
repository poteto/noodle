package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/stamp"
)

func runStampCommand(ctx context.Context, _ *App, _ []Command, args []string) error {
	flags := flag.NewFlagSet("stamp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	outputPath := flags.String("output", "", "Output path for stamped NDJSON")
	flags.StringVar(outputPath, "o", "", "Output path for stamped NDJSON")
	eventsPath := flags.String("events", "", "Optional output path for canonical sidecar events")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if *outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	var eventsWriter io.WriteCloser
	var err error
	if *eventsPath != "" {
		eventsWriter, err = openWritableFile(*eventsPath)
		if err != nil {
			return err
		}
		defer eventsWriter.Close()
	}

	stampedFile, err := openWritableFile(*outputPath)
	if err != nil {
		return err
	}
	defer stampedFile.Close()

	processor := stamp.NewProcessor()
	return processor.Process(ctx, os.Stdin, stampedFile, eventsWriter)
}

func openWritableFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create parent directory for %s: %w", path, err)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return file, nil
}
