//go:build !darwin

package foreground

import "context"

// No bounded foreground-group enumeration is implemented on these platforms.
// In particular, Linux's live per-thread children lists cannot establish group
// membership; do not use them to justify a supposedly unambiguous name.
func processName(context.Context, int) string { return "" }
