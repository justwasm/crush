//go:build windows

package cmd

import (
	"os"
	"syscall"
)

// terminationSignals are the signals that mean "stop now, but tidily":
// Ctrl+C, and SIGTERM (Go delivers a closed Windows console as SIGTERM).
func terminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

// subscribeReloadSignals is a no-op on Windows where SIGHUP is unsupported.
func subscribeReloadSignals() (<-chan os.Signal, func()) {
	return nil, func() {}
}
