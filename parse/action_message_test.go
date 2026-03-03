package parse

import "testing"

func TestParseActionMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		wantTool string
		wantBody string
	}{
		{name: "bash", message: "$ go test ./...", wantTool: "Bash", wantBody: "go test ./..."},
		{name: "think", message: "text:plan next step", wantTool: "Think", wantBody: "plan next step"},
		{name: "prompt", message: "user:fix this bug", wantTool: "Prompt", wantBody: "fix this bug"},
		{name: "tool and body", message: "Read /tmp/file.txt", wantTool: "Read", wantBody: "/tmp/file.txt"},
		{name: "tool only", message: "Skill", wantTool: "Skill", wantBody: ""},
		{name: "empty", message: "   ", wantTool: "", wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseActionMessage(tt.message)
			if got.Tool != tt.wantTool || got.Summary != tt.wantBody {
				t.Fatalf("ParseActionMessage(%q) = tool=%q summary=%q, want tool=%q summary=%q",
					tt.message, got.Tool, got.Summary, tt.wantTool, tt.wantBody)
			}
		})
	}
}
