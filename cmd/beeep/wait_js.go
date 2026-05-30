//go:build js

package main

import "time"

// wait sleeps briefly so that the browser Notification API has time to
// dispatch the notification before the wasm process exits. The beeep
// js implementation may call Notification.requestPermission which is
// asynchronous; a short pause is enough for most cases.
func wait() {
	time.Sleep(time.Second)
}
