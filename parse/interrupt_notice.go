package parse

import (
	"strings"

	"github.com/poteto/noodle/internal/stringx"
)

func isInterruptNotice(text string) bool {
	normalized := stringx.Normalize(strings.Trim(text, " \t\r\n.!?"))
	return normalized == "request interrupted by user"
}
