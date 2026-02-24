package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/stamp"
	"github.com/spf13/cobra"
)

func newStampCmd(_ *App) *cobra.Command {
	var outputPath, eventsPath string
	cmd := &cobra.Command{
		Use:   "stamp",
		Short: cmdmeta.Short("stamp"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStamp(cmd.Context(), outputPath, eventsPath)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for stamped NDJSON")
	cmd.Flags().StringVar(&eventsPath, "events", "", "Optional output path for canonical sidecar events")
	return cmd
}

func runStamp(ctx context.Context, outputPath, eventsPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	var eventsWriter io.WriteCloser
	var err error
	if eventsPath != "" {
		eventsWriter, err = openWritableFile(eventsPath)
		if err != nil {
			return err
		}
		defer eventsWriter.Close()
	}

	stampedFile, err := openWritableFile(outputPath)
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
