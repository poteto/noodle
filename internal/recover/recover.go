package recover

import (
	"fmt"
	"regexp"
	"strings"
)

var recoverySuffixRegexp = regexp.MustCompile(`-recover-(\d+)$`)

// RecoveryChainLength returns retry depth encoded in a session name.
func RecoveryChainLength(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	matches := recoverySuffixRegexp.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0
	}
	var value int
	_, _ = fmt.Sscanf(matches[1], "%d", &value)
	if value < 0 {
		return 0
	}
	return value
}
