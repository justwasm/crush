//go:build !windows && !(js && wasm)

package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

// terminationSignals are the signals that mean "stop now, but tidily":
// Ctrl+C, SIGTERM from a service manager, and SIGHUP from a closed terminal.
// The server reads SIGHUP as a config reload instead; see awaitStop.
func terminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}

// subscribeReloadSignals returns a channel that receives SIGHUP for config
// reload, and a cleanup function to stop receiving signals.
func subscribeReloadSignals() (<-chan os.Signal, func()) {
	signal.Reset(syscall.SIGHUP)
	hupch := make(chan os.Signal, 1)
	signal.Notify(hupch, syscall.SIGHUP)
	return hupch, func() {
		signal.Stop(hupch)
	}
}
