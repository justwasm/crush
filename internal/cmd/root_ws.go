package cmd

import (
	"os"
	"strings"
)

// isWSTransport reports whether WebSocket transport is enabled by environment variable
// (CRUSH_TRANSPORT=ws) or by a ws:// or wss:// host URL scheme.
func isWSTransport(host string) bool {
	if v := os.Getenv("CRUSH_TRANSPORT"); strings.EqualFold(v, "ws") ||
		strings.EqualFold(v, "websocket") || v == "1" || strings.EqualFold(v, "true") {
		return true
	}
	return strings.HasPrefix(host, "ws://") || strings.HasPrefix(host, "wss://")
}
