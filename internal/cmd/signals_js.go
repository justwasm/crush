//go:build js && wasm

package cmd

import (
	"os"
	"syscall"
)

// terminationSignals returns the signals that request tidy termination.
func terminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

// subscribeReloadSignals is a no-op on js/wasm where SIGHUP is unsupported.
func subscribeReloadSignals() (<-chan os.Signal, func()) {
	return nil, func() {}
}
