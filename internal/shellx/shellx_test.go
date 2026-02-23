package shellx

import "testing"

func TestQuote(t *testing.T) {
	if got := Quote("hello world"); got != "'hello world'" {
		t.Fatalf("quote = %q", got)
	}
	if got := Quote("it's"); got != "'it'\"'\"'s'" {
		t.Fatalf("quote escape = %q", got)
	}
}

func TestSanitizeToken(t *testing.T) {
	if got := SanitizeToken("  My Session/42  ", "cook"); got != "my-session-42" {
		t.Fatalf("sanitize = %q", got)
	}
	if got := SanitizeToken("***", "cook"); got != "cook" {
		t.Fatalf("fallback = %q", got)
	}
}
