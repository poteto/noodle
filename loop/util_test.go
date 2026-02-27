package loop

import "testing"

func TestTitleFromPrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		maxWords int
		want     string
	}{
		{"empty", "", 8, ""},
		{"fewer words than max", "Fix the tests", 8, "Fix the tests"},
		{"exactly max words", "one two three four five six seven eight", 8, "one two three four five six seven eight"},
		{"more words than max", "one two three four five six seven eight nine ten", 8, "one two three four five six seven eight..."},
		{"single word", "refactor", 8, "refactor"},
		{"extra whitespace", "  Fix   the   tests  ", 8, "Fix the tests"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleFromPrompt(tt.prompt, tt.maxWords)
			if got != tt.want {
				t.Fatalf("titleFromPrompt(%q, %d) = %q, want %q", tt.prompt, tt.maxWords, got, tt.want)
			}
		})
	}
}

func TestBuildCookPromptIncludesPrompt(t *testing.T) {
	stage := Stage{
		Prompt: "Refactor the auth module to use JWT tokens",
	}
	got := buildCookPrompt("execute-123", stage, nil, "", "", "")
	if want := "Refactor the auth module to use JWT tokens"; !containsSubstring(got, want) {
		t.Fatalf("buildCookPrompt did not include prompt text\ngot: %q", got)
	}
}

func TestBuildCookPromptWithoutPrompt(t *testing.T) {
	stage := Stage{}
	got := buildCookPrompt("execute-123", stage, nil, "", "", "")
	if containsSubstring(got, "\n\n\n") {
		t.Fatalf("buildCookPrompt has double blank lines when prompt is empty\ngot: %q", got)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && len(sub) > 0 && searchSubstring(s, sub)))
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
