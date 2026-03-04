package dispatcher

import (
	"context"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

// processSession implements Session backed by a direct child process.
// It runs the stamp processor in-process (like spritesSession) rather than
// as a shell pipeline stage.
type processSession struct {
	sessionBase
	process    *ProcessHandle
	controller *claudeController // nil for non-steerable sessions
	warnings   []string
}

type processSessionConfig struct {
	id            string
	process       *ProcessHandle
	eventWriter   *event.EventWriter
	canonicalPath string
	stampedPath   string
	prompt        string
	warnings      []string
	controller    *claudeController // nil for non-steerable sessions
	sink          SessionEventSink
}

func newProcessSession(cfg processSessionConfig) *processSession {
	return &processSession{
		sessionBase: newSessionBase(sessionBaseConfig{
			id:            cfg.id,
			eventWriter:   cfg.eventWriter,
			stampedPath:   cfg.stampedPath,
			canonicalPath: cfg.canonicalPath,
			prompt:        cfg.prompt,
			sink:          cfg.sink,
		}),
		process:    cfg.process,
		controller: cfg.controller,
		warnings:   append([]string(nil), cfg.warnings...),
	}
}

func (s *processSession) start(ctx context.Context) {
	s.writeInitialHeartbeat()
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		defer s.closeStreamDone()
		interceptor := &canonicalLineInterceptor{onLine: func(line []byte) {
			s.consumeCanonicalLine(line, s.processHook)
		}}
		s.processStream(ctx, s.process.Stdout(), interceptor)
	}()

	go func() {
		defer s.wg.Done()
		s.waitForExit(ctx)
	}()

	go s.closeEventsWhenDone()

	for _, warning := range s.warnings {
		s.publish(SessionEvent{
			Type:      "warning",
			Message:   warning,
			Timestamp: nowUTC(),
		})
	}
}

func (s *processSession) processHook(ce parse.CanonicalEvent) {
	s.observeCanonicalEvent(ce)
	if s.controller != nil {
		s.controller.NotifyEvent(string(ce.Type))
	}
}

func (s *processSession) waitForExit(ctx context.Context) {
	select {
	case <-s.process.Done():
	case <-ctx.Done():
		_ = s.process.ForceKill()
		<-s.process.Done()
	}

	exitCode, _ := s.process.ExitCode()
	s.resolveAndMarkDone(exitCode, ctx.Err() != nil)
}

func (s *processSession) Terminate() error {
	return s.process.Terminate()
}

func (s *processSession) ForceKill() error {
	err := s.process.ForceKill()
	if err != nil {
		return err
	}
	s.markDone("killed")
	return nil
}

func (s *processSession) Controller() AgentController {
	if s.controller != nil {
		return s.controller
	}
	return noopController{}
}

var _ Session = (*processSession)(nil)
