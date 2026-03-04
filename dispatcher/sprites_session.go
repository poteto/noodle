package dispatcher

import (
	"context"
	"errors"
	"io"
	"time"

	sprites "github.com/superfly/sprites-go"

	"github.com/poteto/noodle/event"
)

type spritesSessionConfig struct {
	id            string
	sprite        spriteHandle
	spriteName    string
	cmd           *sprites.Cmd
	stdout        io.ReadCloser
	runtimeDir    string
	stampedPath   string
	canonicalPath string
	eventWriter   *event.EventWriter
	prompt        string
	warnings      []string
	remoteURL     string
	sink          SessionEventSink
}

type spritesSession struct {
	sessionBase
	sprite     spriteHandle
	spriteName string
	cmd        *sprites.Cmd
	stdout     io.ReadCloser
	runtimeDir string
	remoteURL  string

	startWarnings []string
}

func newSpritesSession(cfg spritesSessionConfig) *spritesSession {
	return &spritesSession{
		sessionBase: newSessionBase(sessionBaseConfig{
			id:            cfg.id,
			eventWriter:   cfg.eventWriter,
			stampedPath:   cfg.stampedPath,
			canonicalPath: cfg.canonicalPath,
			prompt:        cfg.prompt,
			sink:          cfg.sink,
		}),
		sprite:        cfg.sprite,
		spriteName:    cfg.spriteName,
		cmd:           cfg.cmd,
		stdout:        cfg.stdout,
		runtimeDir:    cfg.runtimeDir,
		remoteURL:     cfg.remoteURL,
		startWarnings: append([]string(nil), cfg.warnings...),
	}
}

func (s *spritesSession) start(ctx context.Context) {
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		defer s.closeStreamDone()
		interceptor := &canonicalLineInterceptor{onLine: func(line []byte) {
			s.consumeCanonicalLine(line, s.observeCanonicalEvent)
		}}
		s.processStream(ctx, s.stdout, interceptor)
	}()

	go func() {
		defer s.wg.Done()
		s.waitAndSync(ctx)
	}()

	go s.closeEventsWhenDone()

	for _, warning := range s.startWarnings {
		s.publish(SessionEvent{
			Type:      "warning",
			Message:   warning,
			Timestamp: nowUTC(),
		})
	}
}

func (s *spritesSession) waitAndSync(ctx context.Context) {
	err := s.cmd.Wait()

	exitCode := 0
	shouldSyncBack := true
	if err != nil {
		var exitErr *sprites.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
			if ctx.Err() != nil {
				shouldSyncBack = false
			}
		}
	}

	// Push changes from sprite back to git remote.
	if s.remoteURL != "" && shouldSyncBack {
		syncCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, pushErr := pushChangesFromSprite(syncCtx, s.sprite, s.id)
		if pushErr != nil {
			s.publish(SessionEvent{
				Type:      "warning",
				Message:   "sync-back push failed: " + pushErr.Error(),
				Timestamp: nowUTC(),
			})
		} else if result.Type == SyncResultTypeBranch {
			if writeErr := writeSyncResult(s.runtimeDir, s.id, result); writeErr != nil {
				s.publish(SessionEvent{
					Type:      "warning",
					Message:   "write sync result: " + writeErr.Error(),
					Timestamp: nowUTC(),
				})
			}
		}
	}

	s.resolveAndMarkDone(exitCode, ctx.Err() != nil)
}

func (s *spritesSession) Terminate() error {
	if s.cmd != nil {
		_ = s.cmd.Signal("SIGTERM")
	}
	return nil
}

func (s *spritesSession) ForceKill() error {
	if s.cmd != nil {
		_ = s.cmd.Signal("SIGKILL")
	}
	s.markDone("killed")
	return nil
}

func (s *spritesSession) Controller() AgentController { return noopController{} }

var _ Session = (*spritesSession)(nil)
