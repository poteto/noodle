package adapter

import "encoding/json"

// Well-known status values. Not exhaustive — adapters may return any string.
const (
	BacklogStatusOpen       = "open"
	BacklogStatusInProgress = "in_progress"
	BacklogStatusDone       = "done"
)

// Known field keys — used to separate known from extra fields during unmarshal.
var knownBacklogKeys = map[string]bool{
	"id": true, "title": true, "status": true, "plan": true,
}

type BacklogItem struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Status string         `json:"status,omitempty"`
	Plan   string         `json:"plan,omitempty"`
	Extra  map[string]any `json:"-"`
}

func (b *BacklogItem) UnmarshalJSON(data []byte) error {
	// Unmarshal known fields via an alias to avoid infinite recursion.
	type plain BacklogItem
	if err := json.Unmarshal(data, (*plain)(b)); err != nil {
		return err
	}

	// Capture all fields into a map, then extract unknown ones.
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for key, raw := range all {
		if knownBacklogKeys[key] {
			continue
		}
		if b.Extra == nil {
			b.Extra = make(map[string]any, len(all)-len(knownBacklogKeys))
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		b.Extra[key] = v
	}
	return nil
}

func (b BacklogItem) MarshalJSON() ([]byte, error) {
	type plain BacklogItem
	base, err := json.Marshal((plain)(b))
	if err != nil {
		return nil, err
	}
	if len(b.Extra) == 0 {
		return base, nil
	}
	// Merge extra fields into the base object.
	extra, err := json.Marshal(b.Extra)
	if err != nil {
		return nil, err
	}
	// Splice: strip trailing } from base, strip leading { from extra, join with comma.
	base[len(base)-1] = ','
	return append(base, extra[1:]...), nil
}

