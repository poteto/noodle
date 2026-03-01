package loop

import (
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/skill"
	"github.com/poteto/noodle/worktree"
)

type noOpWorktree struct{}

func (noOpWorktree) Create(string) error            { return nil }
func (noOpWorktree) Merge(string, string) error     { return nil }
func (noOpWorktree) MergeRemoteBranch(string) error { return nil }
func (noOpWorktree) Cleanup(string, bool) error     { return nil }

func defaultDependencies(projectDir, runtimeDir, noodleBin string, cfg config.Config, logger *slog.Logger, sink dispatcher.SessionEventSink) Dependencies {
	resolver := skill.Resolver{SearchPaths: cfg.Skills.Paths}
	local := dispatcher.NewProcessDispatcher(dispatcher.ProcessDispatcherConfig{
		ProjectDir:    projectDir,
		RuntimeDir:    runtimeDir,
		NoodleBin:     noodleBin,
		SkillResolver: resolver,
		RuntimeKind:   "process",
		Sink:          sink,
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
	runtimes := map[string]loopruntime.Runtime{
		"process": loopruntime.NewProcessRuntime(local, runtimeDir, cfg.Runtime.Process.MaxConcurrent),
	}
	if runtimeEnabled(cfg.AvailableRuntimes(), "sprites") {
		spriteName := strings.TrimSpace(cfg.Runtime.Sprites.SpriteName)
		if spriteName == "" {
			logger.Warn("sprites runtime unavailable: sprite_name not set")
		} else {
			sd := dispatcher.NewSpritesDispatcher(dispatcher.SpritesDispatcherConfig{
				ProjectDir:    projectDir,
				RuntimeDir:    runtimeDir,
				NoodleBin:     noodleBin,
				SkillResolver: resolver,
				SpriteName:    spriteName,
				Token:         cfg.Runtime.Sprites.Token(),
				GitToken:      cfg.Runtime.Sprites.GitToken(),
				Sink:          sink,
			})
			r := loopruntime.NewDispatcherRuntime("sprites", sd, runtimeDir)
			r.SetMaxConcurrent(cfg.Runtime.Sprites.MaxConcurrent)
			runtimes["sprites"] = r
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
		Runtimes:       runtimes,
		Worktree:       wt,
		Adapter:        adapter.NewRunner(projectDir, cfg),
		Mise:           mise.NewBuilder(projectDir, cfg),
		Monitor:        monitor.NewMonitor(runtimeDir),
		Now:            time.Now,
		OrdersFile:     filepath.Join(runtimeDir, "orders.json"),
		OrdersNextFile: filepath.Join(runtimeDir, "orders-next.json"),
		StatusFile:     filepath.Join(runtimeDir, "status.json"),
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
