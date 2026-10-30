//go:build js && wasm

package cmd

// detachStderr is a no-op on js/wasm: there is no child process fork.
func detachStderr() (restore func(), err error) {
	return func() {}, nil
}
