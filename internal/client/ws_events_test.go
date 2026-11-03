package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestClient_IsWSTransport(t *testing.T) {
	c, err := NewClient(t.TempDir(), "tcp", "127.0.0.1:8080")
	require.NoError(t, err)

	require.False(t, c.IsWSTransport())

	c.SetTransport("ws")
	require.True(t, c.IsWSTransport())
	require.Equal(t, "ws", c.Transport())

	c.SetTransport("")
	require.False(t, c.IsWSTransport())

	t.Setenv("CRUSH_TRANSPORT", "ws")
	require.True(t, c.IsWSTransport())

	t.Setenv("CRUSH_TRANSPORT", "websocket")
	require.True(t, c.IsWSTransport())

	t.Setenv("CRUSH_TRANSPORT", "1")
	require.True(t, c.IsWSTransport())

	t.Setenv("CRUSH_TRANSPORT", "true")
	require.True(t, c.IsWSTransport())
}

func TestSubscribeEventsWS_ReceivesEvents(t *testing.T) {
	t.Parallel()

	firstEventSent := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()

		// Write a host theme event
		themePayload := proto.HostThemeState{
			WorkspaceID: "ws1",
			Theme: &proto.HostTheme{
				Name: "catppuccin",
			},
		}
		data, err := json.Marshal(pubsub.Payload{
			Type:    pubsub.PayloadTypeHostTheme,
			Payload: json.RawMessage(mustJSON(t, pubsub.Event[proto.HostThemeState]{Type: pubsub.UpdatedEvent, Payload: themePayload})),
		})
		require.NoError(t, err)
		require.NoError(t, conn.Write(ctx, websocket.MessageText, data))

		// Write a message event
		msgPayload := proto.Message{
			ID:        "msg-1",
			SessionID: "s-1",
			Role:      proto.Assistant,
		}
		msgData, err := json.Marshal(pubsub.Payload{
			Type:    pubsub.PayloadTypeMessage,
			Payload: json.RawMessage(mustJSON(t, pubsub.Event[proto.Message]{Type: pubsub.CreatedEvent, Payload: msgPayload})),
		})
		require.NoError(t, err)
		require.NoError(t, conn.Write(ctx, websocket.MessageText, msgData))

		close(firstEventSent)

		// Keep connection open until client closes or context ends
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := captureClient(t, srv)
	c.SetTransport("ws")

	events, err := c.SubscribeEvents(ctx, "ws1")
	require.NoError(t, err)

	select {
	case <-firstEventSent:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server event write")
	}

	ev1 := <-events
	themeEvent, ok := ev1.(pubsub.Event[proto.HostThemeState])
	require.True(t, ok, "expected HostThemeState, got %T", ev1)
	require.Equal(t, "catppuccin", themeEvent.Payload.Theme.Name)

	ev2 := <-events
	msgEvent, ok := ev2.(pubsub.Event[proto.Message])
	require.True(t, ok, "expected Message, got %T", ev2)
	require.Equal(t, "msg-1", msgEvent.Payload.ID)
}

func TestSubscribeEventsWS_ContextCancelClosesEvents(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	c := captureClient(t, srv)
	events, err := c.SubscribeEventsWS(ctx, "ws1")
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-events:
		require.False(t, ok, "events channel should be closed after cancel")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event channel close")
	}
}

func TestSubscribeEventsWS_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c := captureClient(t, srv)
	_, err := c.SubscribeEventsWS(ctx, "nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

func TestSubscribeEventsWS_UnixSocket(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join("/tmp", fmt.Sprintf("test-ws-%d.sock", time.Now().UnixNano()))
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	defer ln.Close()

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			msgPayload := proto.Message{
				ID:        "msg-unix-1",
				SessionID: "s-unix",
				Role:      proto.Assistant,
			}
			msgData, err := json.Marshal(pubsub.Payload{
				Type:    pubsub.PayloadTypeMessage,
				Payload: json.RawMessage(mustJSON(t, pubsub.Event[proto.Message]{Type: pubsub.CreatedEvent, Payload: msgPayload})),
			})
			if err != nil {
				return
			}
			_ = conn.Write(r.Context(), websocket.MessageText, msgData)

			for {
				if _, _, err := conn.Read(r.Context()); err != nil {
					return
				}
			}
		}),
	}
	go srv.Serve(ln)
	defer srv.Close()

	c, err := NewClient(t.TempDir(), "unix", sockPath)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := c.SubscribeEventsWS(ctx, "ws1")
	require.NoError(t, err)

	select {
	case ev := <-events:
		msgEvent, ok := ev.(pubsub.Event[proto.Message])
		require.True(t, ok)
		require.Equal(t, "msg-unix-1", msgEvent.Payload.ID)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event over unix socket")
	}
}
