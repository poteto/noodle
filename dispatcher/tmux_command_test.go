package dispatcher

import (
	"strings"
	"testing"
)

func TestBuildPipelineCommand(t *testing.T) {
	command := buildPipelineCommand(
		"'claude' -p < '/tmp/prompt.txt' 2>&1",
		"/usr/local/bin/noodle",
		"/tmp/stamped.ndjson",
		"/tmp/canonical.ndjson",
	)
	if !strings.Contains(command, "stamp --output '/tmp/stamped.ndjson' --events '/tmp/canonical.ndjson'") {
		t.Fatalf("unexpected pipeline command: %s", command)
	}
}
