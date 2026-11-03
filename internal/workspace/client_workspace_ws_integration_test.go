package workspace_test

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/workspace"
	"github.com/stretchr/testify/require"
)

func TestClientWorkspace_WebSocketEndToEnd(t *testing.T) {
	xdgIsolate(t)
	rt := newRuntimeServer(t)

	cwd := t.TempDir()
	dataDir := t.TempDir()

	c := rt.newClient(t, cwd)
	c.SetTransport("ws")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	wsProto, err := c.CreateWorkspace(ctx, proto.Workspace{Path: cwd, DataDir: dataDir})
	require.NoError(t, err)

	ws := workspace.NewClientWorkspace(c, *wsProto)
	t.Cleanup(ws.Shutdown)

	events := make(chan tea.Msg, 100)
	go ws.RunSubscriptionForTest(func(msg tea.Msg) {
		events <- msg
	})

	// Wait briefly for WebSocket connection to establish
	time.Sleep(100 * time.Millisecond)

	// Create a session via ClientWorkspace
	createdSess, err := ws.CreateSession(ctx, "WebSocket Session Test")
	require.NoError(t, err)
	require.NotEmpty(t, createdSess.ID)
	require.Equal(t, "WebSocket Session Test", createdSess.Title)

	// Verify that the session created event was delivered over WebSocket
	select {
	case msg := <-events:
		ev, ok := msg.(pubsub.Event[session.Session])
		require.True(t, ok, "expected pubsub.Event[session.Session], got %T", msg)
		require.Equal(t, createdSess.ID, ev.Payload.ID)
		require.Equal(t, "WebSocket Session Test", ev.Payload.Title)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for session created event over WebSocket")
	}

	// Mutate config (SetCompactMode) via ClientWorkspace
	require.NoError(t, ws.SetCompactMode(config.ScopeGlobal, true))

	// Verify Config snapshot updated
	require.True(t, compactMode(ws.Config()), "compact mode should be enabled after SetCompactMode")
}
