package mise

import (
	"github.com/poteto/noodle/internal/jsonx"
)

func writeBriefAtomic(path string, brief Brief) error {
	return jsonx.WriteJSONIndented(path, brief)
}
