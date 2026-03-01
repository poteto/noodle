package dispatcher

import "strings"

// ProcessStartError marks a failure to launch a local process runtime session.
type ProcessStartError struct {
	Cause error
}

func (e ProcessStartError) Error() string {
	if e.Cause == nil {
		return "process start failed"
	}
	return "process start failed: " + strings.TrimSpace(e.Cause.Error())
}

func (e ProcessStartError) Unwrap() error {
	return e.Cause
}
