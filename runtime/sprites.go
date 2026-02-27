package runtime

import "github.com/poteto/noodle/dispatcher"

// NewSpritesRuntime adapts the sprites dispatcher to the Runtime interface.
func NewSpritesRuntime(d dispatcher.Dispatcher, runtimeDir string, maxConcurrent int) Runtime {
	r := NewDispatcherRuntime("sprites", d, runtimeDir)
	r.SetMaxConcurrent(maxConcurrent)
	return r
}
