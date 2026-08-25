//go:build js

package cmd

import (
	"cmp"
	"os"
)

// resolveCurrentUser returns the current user's username and home directory.
//
// On js/wasm, user.Current() always fails because cgo is unavailable and
// /etc/passwd doesn't exist in the sandbox. Fall back to $USER and $HOME,
// which are only used downstream for HTML display and ~/ path shortening,
// defaulting to DefaultUser and DefaultHome when the runtime does not set
// them.
func resolveCurrentUser() (username, homeDir string) {
	return cmp.Or(os.Getenv("USER"), DefaultUser), cmp.Or(os.Getenv("HOME"), DefaultHome)
}
