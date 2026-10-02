package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
)

// backlogAdapterTimeout bounds one backlog adapter call made for a control
// command. Controls run inside the loop cycle, so a hung script must not stall
// it.
const backlogAdapterTimeout = 30 * time.Second

// controlBacklogAdd adds an item through the backlog adapter's add script. The
// item reaches the snapshot on the next mise sync, which runs in the same
// cycle.
func (l *Loop) controlBacklogAdd(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("backlog item title empty")
	}
	payload, err := backlogPayload(title)
	if err != nil {
		return err
	}
	return l.runBacklogAdapter("add", adapter.RunOptions{Stdin: payload})
}

// controlBacklogEdit retitles an item through the backlog adapter's edit
// script.
func (l *Loop) controlBacklogEdit(id, title string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("backlog item id empty")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("backlog item title empty")
	}
	payload, err := backlogPayload(title)
	if err != nil {
		return err
	}
	return l.runBacklogAdapter("edit", adapter.RunOptions{Args: []string{id}, Stdin: payload})
}

// controlBacklogDone marks an item done through the backlog adapter's done
// script — the same script the loop runs when an order for the item merges.
func (l *Loop) controlBacklogDone(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("backlog item id empty")
	}
	return l.runBacklogAdapter("done", adapter.RunOptions{Args: []string{id}})
}

// backlogPayload encodes the {"title": ...} object the add and edit scripts
// read on stdin. HTML escaping is off: json.Marshal would turn "&" into
// "\u0026", which a shell script stores verbatim.
func backlogPayload(title string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]string{"title": title}); err != nil {
		return nil, fmt.Errorf("encode backlog item: %w", err)
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func (l *Loop) runBacklogAdapter(action string, options adapter.RunOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), backlogAdapterTimeout)
	defer cancel()
	_, err := l.deps.Adapter.Run(ctx, "backlog", action, options)
	return err
}

// recordBacklogResult keeps the latest backlog control failure for the
// snapshot and clears it once a later backlog control succeeds.
func (l *Loop) recordBacklogResult(err error) error {
	if err != nil {
		l.lastBacklogError = err.Error()
		return err
	}
	l.lastBacklogError = ""
	return nil
}
