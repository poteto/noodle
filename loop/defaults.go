package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/shellx"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/skill"
	"github.com/poteto/noodle/worktree"
)

type noOpWorktree struct{}

func (noOpWorktree) Create(string) error            { return nil }
func (noOpWorktree) Merge(string) error             { return nil }
func (noOpWorktree) MergeRemoteBranch(string) error { return nil }
func (noOpWorktree) Cleanup(string, bool) error     { return nil }

func defaultDependencies(projectDir, runtimeDir, noodleBin string, cfg config.Config) Dependencies {
	resolver := skill.Resolver{SearchPaths: cfg.Skills.Paths}
	local := dispatcher.NewTmuxDispatcher(dispatcher.TmuxDispatcherConfig{
		ProjectDir:    projectDir,
		RuntimeDir:    runtimeDir,
		NoodleBin:     noodleBin,
		SkillResolver: resolver,
		RuntimeKind:   "tmux",
		ProviderConfigs: dispatcher.ProviderConfigs{
			Claude: dispatcher.ProviderConfig{
				Path: cfg.Agents.Claude.Path,
				Args: cfg.Agents.Claude.Args,
			},
			Codex: dispatcher.ProviderConfig{
				Path: cfg.Agents.Codex.Path,
				Args: cfg.Agents.Codex.Args,
			},
		},
	})
	factory := dispatcher.NewDispatcherFactory()
	if err := factory.Register("tmux", local); err != nil {
		panic(err)
	}
	if runtimeEnabled(cfg.AvailableRuntimes(), "sprites") {
		wrapperDir, err := ensureSpritesProviderWrappers(runtimeDir, cfg.Runtime.Sprites)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: sprites runtime unavailable: %v\n", err)
		} else {
			remoteProviders := dispatcher.ProviderConfigs{
				Claude: dispatcher.ProviderConfig{
					Path: wrapperDir,
					Args: cfg.Agents.Claude.Args,
				},
				Codex: dispatcher.ProviderConfig{
					Path: wrapperDir,
					Args: cfg.Agents.Codex.Args,
				},
			}
			sprites := dispatcher.NewTmuxDispatcher(dispatcher.TmuxDispatcherConfig{
				ProjectDir:      projectDir,
				RuntimeDir:      runtimeDir,
				NoodleBin:       noodleBin,
				SkillResolver:   resolver,
				RuntimeKind:     "sprites",
				ProviderConfigs: remoteProviders,
			})
			if err := factory.Register("sprites", sprites); err != nil {
				panic(err)
			}
		}
	}

	wtApp, _ := worktree.NewApp()
	var wt WorktreeManager
	if wtApp != nil {
		wtApp.CmdPrefix = "noodle worktree"
		wtApp.Quiet = true
		wt = wtApp
	} else {
		wt = noOpWorktree{}
	}
	return Dependencies{
		Dispatcher: factory,
		Worktree:   wt,
		Adapter:    adapter.NewRunner(projectDir, cfg),
		Mise:       mise.NewBuilder(projectDir, cfg),
		Monitor:    monitor.NewMonitor(runtimeDir),
		Now:        time.Now,
		QueueFile:  filepath.Join(runtimeDir, "queue.json"),
	}
}

func runtimeEnabled(available []string, kind string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, runtime := range available {
		if strings.ToLower(strings.TrimSpace(runtime)) == kind {
			return true
		}
	}
	return false
}

func ensureSpritesProviderWrappers(runtimeDir string, cfg config.SpritesConfig) (string, error) {
	spriteName := strings.TrimSpace(cfg.SpriteName)
	if spriteName == "" {
		return "", fmt.Errorf("runtime.sprites.sprite_name not set")
	}

	dir := filepath.Join(runtimeDir, "bin", "sprites")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create sprites wrapper directory: %w", err)
	}

	claudePath := filepath.Join(dir, "claude")
	claudeScript := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"exec sprite -s " + shellx.Quote(spriteName) + " exec claude \"$@\"",
		"",
	}, "\n")
	if err := os.WriteFile(claudePath, []byte(claudeScript), 0o755); err != nil {
		return "", fmt.Errorf("write sprites claude wrapper: %w", err)
	}

	codexPath := filepath.Join(dir, "codex")
	codexScript := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"exec sprite -s " + shellx.Quote(spriteName) + " exec codex \"$@\"",
		"",
	}, "\n")
	if err := os.WriteFile(codexPath, []byte(codexScript), 0o755); err != nil {
		return "", fmt.Errorf("write sprites codex wrapper: %w", err)
	}

	return dir, nil
}
