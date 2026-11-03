package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWSTransport(t *testing.T) {
	require.False(t, isWSTransport("http://127.0.0.1:8080"))
	require.False(t, isWSTransport("unix:///tmp/crush.sock"))

	require.True(t, isWSTransport("ws://127.0.0.1:8080"))
	require.True(t, isWSTransport("wss://example.com:8443"))

	t.Setenv("CRUSH_TRANSPORT", "ws")
	require.True(t, isWSTransport("http://127.0.0.1:8080"))
	require.True(t, isWSTransport("unix:///tmp/crush.sock"))

	t.Setenv("CRUSH_TRANSPORT", "websocket")
	require.True(t, isWSTransport("http://127.0.0.1:8080"))

	t.Setenv("CRUSH_TRANSPORT", "1")
	require.True(t, isWSTransport("http://127.0.0.1:8080"))

	t.Setenv("CRUSH_TRANSPORT", "true")
	require.True(t, isWSTransport("http://127.0.0.1:8080"))
}
