//go:build !js

package main

// wait is a no-op on native platforms: the notification is sent synchronously
// by beeep so there is nothing to wait for.
func wait() {}
