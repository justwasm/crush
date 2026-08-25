package cmd

// DefaultUser and DefaultHome are fallbacks for the current user's username
// and home directory when the OS/user database cannot provide them (e.g. on
// js/wasm, where user.Current() always fails).
const (
	DefaultUser = "user"
	DefaultHome = "home"
)
