//go:build wasm

package notification

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
)

// NativeBackend is a no-op implementation for WASM builds.
// The beeep library used for native OS notifications is incompatible with
// WASM — it calls syscall/js functions that require a JavaScript runtime
// with browser-level notification APIs, which aren't available in the
// WASM environments where Crush runs.
type NativeBackend struct{}

// NewNativeBackend creates a new NativeBackend that silently discards
// notifications. WASM environments lack native OS notification support,
// so all notifications are no-ops. Users should configure an OSC or bell
// backend for notification support in WASM.
func NewNativeBackend(_ []byte) *NativeBackend {
	slog.Debug("NativeBackend is a no-op in WASM builds")
	return &NativeBackend{}
}

// Send silently discards the notification in WASM environments.
func (b *NativeBackend) Send(_ Notification) tea.Cmd {
	slog.Debug("Native notification suppressed in WASM (no beeep support)")
	return nil
}
