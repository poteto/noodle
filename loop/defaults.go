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
		spriteName := strings.TrimSpace(cfg.Runtime.Sprites.SpriteName)
		if spriteName == "" {
			fmt.Fprintf(os.Stderr, "warning: sprites runtime unavailable: sprite_name not set\n")
		} else {
			sd := dispatcher.NewSpritesDispatcher(dispatcher.SpritesDispatcherConfig{
				ProjectDir:    projectDir,
				RuntimeDir:    runtimeDir,
				NoodleBin:     noodleBin,
				SkillResolver: resolver,
				SpriteName:    spriteName,
				Token:         cfg.Runtime.Sprites.Token(),
				GitToken:      cfg.Runtime.Sprites.GitToken(),
			})
			if err := factory.Register("sprites", sd); err != nil {
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
