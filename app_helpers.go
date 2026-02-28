package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/skill"
)

func (a *App) ProjectDir() (string, error) {
	if a.projectDir != "" {
		return a.projectDir, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	return cwd, nil
}

func (a *App) RuntimeDir() (string, error) {
	projectDir, err := a.ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, ".noodle"), nil
}

func (a *App) NoodleBinaryPath() (string, error) {
	noodleBin, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return noodleBin, nil
}

func (a *App) SkillResolver() skill.Resolver {
	if a == nil {
		return skill.Resolver{}
	}
	return skill.Resolver{SearchPaths: a.Config.Skills.Paths}
}

func (a *App) ProviderConfigs() dispatcher.ProviderConfigs {
	if a == nil {
		return dispatcher.ProviderConfigs{}
	}
	return dispatcher.ProviderConfigs{
		Claude: dispatcher.ProviderConfig{
			Path: a.Config.Agents.Claude.Path,
			Args: a.Config.Agents.Claude.Args,
		},
		Codex: dispatcher.ProviderConfig{
			Path: a.Config.Agents.Codex.Path,
			Args: a.Config.Agents.Codex.Args,
		},
	}
}
