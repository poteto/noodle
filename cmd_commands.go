package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func runCommandsCommand(_ context.Context, _ *App, catalog []Command, args []string) error {
	flags := flag.NewFlagSet("commands", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "Output as JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *asJSON {
		infos := make([]commandInfo, 0, len(catalog))
		for _, command := range catalog {
			infos = append(infos, commandInfo{
				Name:        command.Name,
				Description: command.Description,
				Category:    command.Category,
			})
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(infos)
	}

	for _, command := range catalog {
		fmt.Printf("%s\t%s\t%s\n", command.Name, command.Category, command.Description)
	}
	return nil
}

type commandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
