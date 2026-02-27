package runtime

import "github.com/poteto/noodle/dispatcher"

// NewTmuxRuntime adapts the tmux dispatcher to the Runtime interface.
func NewTmuxRuntime(d dispatcher.Dispatcher, runtimeDir string, maxConcurrent int) Runtime {
	r := NewDispatcherRuntime("tmux", d, runtimeDir)
	r.SetMaxConcurrent(maxConcurrent)
	return r
}
