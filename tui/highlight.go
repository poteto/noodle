package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// renderEventBody processes an event body for display: prose is word-wrapped
// to maxWidth, fenced code blocks are syntax-highlighted and passed through
// without wrapping (since they contain ANSI escapes and meaningful whitespace).
func renderEventBody(body string, maxWidth int) []string {
	if !strings.Contains(body, "```") {
		return wrapText(body, maxWidth)
	}

	lines := strings.Split(body, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		lang, ok := parseFenceOpen(lines[i])
		if !ok {
			// Accumulate prose lines until next fence or end.
			var prose []string
			for i < len(lines) {
				if _, fenced := parseFenceOpen(lines[i]); fenced {
					break
				}
				prose = append(prose, lines[i])
				i++
			}
			out = append(out, wrapText(strings.Join(prose, "\n"), maxWidth)...)
			continue
		}

		// Skip opening fence.
		i++

		// Collect code block body until closing fence.
		var codeLines []string
		for i < len(lines) {
			if strings.TrimRight(lines[i], " \t") == "```" {
				break
			}
			codeLines = append(codeLines, lines[i])
			i++
		}
		if i < len(lines) {
			i++ // skip closing ```
		}

		code := strings.Join(codeLines, "\n")
		highlighted := highlightCode(code, lang)
		// Code lines are not word-wrapped — they contain ANSI escapes.
		for _, hl := range strings.Split(highlighted, "\n") {
			out = append(out, hl)
		}
	}

	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// parseFenceOpen checks if a line is a fenced code block opener (```lang).
// Returns the language and true, or ("", false) if not a fence.
func parseFenceOpen(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "```") {
		return "", false
	}
	rest := strings.TrimSpace(trimmed[3:])
	if rest == "" {
		return "", true
	}
	if strings.ContainsAny(rest, " \t") {
		return "", false
	}
	return rest, true
}

func highlightCode(code, language string) string {
	if language == "" {
		return code
	}
	lexer := lexers.Get(language)
	if lexer == nil {
		return code
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return code
	}

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return code
	}

	result := buf.String()
	result = strings.TrimRight(result, "\n")
	return result
}
