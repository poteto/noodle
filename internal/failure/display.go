package failure

// OwnerPrefixForDisplay returns a bracketed owner prefix when the failure
// metadata should be surfaced as an agent-owned mistake in operator-facing
// summaries.
func OwnerPrefixForDisplay(class FailureClass, owner FailureOwner) string {
	if owner == "" || class != FailureClassAgentMistake {
		return ""
	}
	return "[" + string(owner) + "] "
}
