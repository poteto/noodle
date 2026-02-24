package dispatcher

import "strings"

func normalizeRuntime(runtime string) string {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	if runtime == "" {
		return "tmux"
	}
	return runtime
}
