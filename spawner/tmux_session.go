package spawner

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/poteto/noodle/parse"
)

type tmuxSession struct {
	id            string
	tmuxName      string
	worktreePath  string
	env           []string
	canonicalPath string
	run           commandRunner

	mu       sync.Mutex
	status   string
	costUSD  float64
	done     chan struct{}
	doneOnce sync.Once
	wg       sync.WaitGroup
	eventsMu sync.Once
	events   chan SessionEvent

	startWarnings []string
}

func newTmuxSession(
	id string,
	tmuxName string,
	worktreePath string,
	env []string,
	canonicalPath string,
	startWarnings []string,
	run commandRunner,
) *tmuxSession {
	return &tmuxSession{
		id:            id,
		tmuxName:      tmuxName,
		worktreePath:  worktreePath,
		env:           append([]string(nil), env...),
		canonicalPath: canonicalPath,
		run:           run,
		status:        "running",
		done:          make(chan struct{}),
		events:        make(chan SessionEvent, 32),
		startWarnings: append([]string(nil), startWarnings...),
	}
}

func (s *tmuxSession) start(ctx context.Context) {
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.monitorPane(ctx)
	}()
	go func() {
		defer s.wg.Done()
		s.monitorCanonicalEvents(ctx)
	}()
	go s.closeEventsWhenDone()
	if len(s.startWarnings) > 0 {
		for _, warning := range s.startWarnings {
			s.publish(SessionEvent{
				Type:      "warning",
				Message:   warning,
				Timestamp: nowUTC(),
			})
		}
	}
}

func (s *tmuxSession) ID() string {
	return s.id
}

func (s *tmuxSession) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *tmuxSession) Events() <-chan SessionEvent {
	return s.events
}

func (s *tmuxSession) Done() <-chan struct{} {
	return s.done
}

func (s *tmuxSession) TotalCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.costUSD
}

func (s *tmuxSession) Kill() error {
	_, _ = s.run(context.Background(), s.worktreePath, s.env, "tmux", "kill-session", "-t", s.tmuxName)
	s.markDone("killed")
	return nil
}

func (s *tmuxSession) monitorPane(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.markDone("cancelled")
			return
		case <-s.done:
			return
		case <-ticker.C:
			if _, err := s.run(ctx, s.worktreePath, s.env, "tmux", "has-session", "-t", s.tmuxName); err != nil {
				s.markDone("completed")
				return
			}
		}
	}
}

func (s *tmuxSession) monitorCanonicalEvents(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	seen := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			events, err := readCanonicalEvents(s.canonicalPath)
			if err != nil {
				continue
			}
			if seen > len(events) {
				seen = 0
			}
			for _, event := range events[seen:] {
				s.consumeCanonical(event)
			}
			seen = len(events)
		}
	}
}

func readCanonicalEvents(path string) ([]parse.CanonicalEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 64<<20)
	events := make([]parse.CanonicalEvent, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *tmuxSession) consumeCanonical(event parse.CanonicalEvent) {
	if event.CostUSD > 0 {
		s.mu.Lock()
		s.costUSD += event.CostUSD
		s.mu.Unlock()
	}

	message := strings.TrimSpace(event.Message)
	if message == "" {
		message = string(event.Type)
	}
	s.publish(SessionEvent{
		Type:      string(event.Type),
		Message:   message,
		Timestamp: event.Timestamp,
		CostUSD:   event.CostUSD,
		TokensIn:  event.TokensIn,
		TokensOut: event.TokensOut,
	})
}

func (s *tmuxSession) publish(event SessionEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = nowUTC()
	}

	select {
	case <-s.done:
		return
	default:
	}

	select {
	case s.events <- event:
		return
	default:
		// Keep stream flowing under burst output by dropping one oldest event.
		select {
		case <-s.events:
		default:
		}
		select {
		case <-s.done:
			return
		default:
		}
		select {
		case s.events <- event:
		default:
		}
	}
}

func (s *tmuxSession) markDone(status string) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.status = status
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *tmuxSession) closeEventsWhenDone() {
	<-s.done
	s.wg.Wait()
	s.eventsMu.Do(func() {
		close(s.events)
	})
}
