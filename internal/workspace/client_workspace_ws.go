package workspace

import (
	"context"
	"os"
	"strings"
)

// isWSTransport reports whether WebSocket transport is enabled for this workspace.
func (w *ClientWorkspace) isWSTransport() bool {
	if w.client != nil && w.client.IsWSTransport() {
		return true
	}
	v := os.Getenv("CRUSH_TRANSPORT")
	return strings.EqualFold(v, "ws") || strings.EqualFold(v, "websocket") || v == "1" || strings.EqualFold(v, "true")
}

// subscribeEvents subscribes to workspace events using WebSocket when configured,
// or falls back to SSE.
func (w *ClientWorkspace) subscribeEvents(ctx context.Context, workspaceID string) (<-chan any, error) {
	if w.isWSTransport() {
		return w.client.SubscribeEventsWS(ctx, workspaceID)
	}
	return w.client.SubscribeEvents(ctx, workspaceID)
}
