//go:build !js

package cmd

import (
	"cmp"
	"os/user"
)

// resolveCurrentUser returns the current user's username and home directory.
//
// On platforms with cgo or a real user database this is a thin wrapper
// around user.Current(), defaulting to DefaultUser and DefaultHome on
// failure.
func resolveCurrentUser() (username, homeDir string) {
	u, err := user.Current()
	if err != nil {
		u = &user.User{}
	}
	return cmp.Or(u.Username, DefaultUser), cmp.Or(u.HomeDir, DefaultHome)
}
