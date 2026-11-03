package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/client"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestClientWorkspace_IsWSTransport(t *testing.T) {
	c, err := client.NewClient(t.TempDir(), "tcp", "127.0.0.1:8080")
	require.NoError(t, err)

	ws := NewClientWorkspace(c, proto.Workspace{ID: "ws-1"})
	defer ws.Shutdown()

	require.False(t, ws.isWSTransport())

	c.SetTransport("ws")
	require.True(t, ws.isWSTransport())

	c.SetTransport("")
	require.False(t, ws.isWSTransport())

	t.Setenv("CRUSH_TRANSPORT", "ws")
	require.True(t, ws.isWSTransport())
}

func TestClientWorkspace_WebSocketSubscription(t *testing.T) {
	var mu sync.Mutex
	connCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connCount++
		mu.Unlock()

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		ctx := r.Context()

		// Write a message event
		msgPayload := proto.Message{
			ID:        "msg-ws-1",
			SessionID: "sess-1",
			Role:      proto.Assistant,
			Parts: []proto.ContentPart{
				proto.TextContent{Text: "Streamed via WebSocket"},
			},
		}
		rawMsg, _ := json.Marshal(pubsub.Event[proto.Message]{Type: pubsub.CreatedEvent, Payload: msgPayload})
		envData, _ := json.Marshal(pubsub.Payload{
			Type:    pubsub.PayloadTypeMessage,
			Payload: json.RawMessage(rawMsg),
		})

		_ = conn.Write(ctx, websocket.MessageText, envData)

		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	c, err := client.NewClient(t.TempDir(), "tcp", u.Host)
	require.NoError(t, err)
	c.SetTransport("ws")

	ws := NewClientWorkspace(c, proto.Workspace{ID: "ws-1"})
	defer ws.Shutdown()

	received := make(chan tea.Msg, 10)
	go ws.runSubscription(func(msg tea.Msg) {
		received <- msg
	})

	select {
	case msg := <-received:
		ev, ok := msg.(pubsub.Event[message.Message])
		require.True(t, ok, "expected pubsub.Event[message.Message], got %T", msg)
		require.Equal(t, "msg-ws-1", ev.Payload.ID)
		require.Len(t, ev.Payload.Parts, 1)
		textPart, ok := ev.Payload.Parts[0].(message.TextContent)
		require.True(t, ok)
		require.Equal(t, "Streamed via WebSocket", textPart.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message over WebSocket subscription")
	}

	mu.Lock()
	require.GreaterOrEqual(t, connCount, 1)
	mu.Unlock()
}

func TestClientWorkspace_WebSocketReconnect(t *testing.T) {
	var mu sync.Mutex
	connCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock host theme endpoint which afterReconnect calls
		if r.URL.Path == "/v1/workspaces/ws-1/theme" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(proto.HostThemeState{})
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		mu.Lock()
		connCount++
		currentConn := connCount
		mu.Unlock()

		ctx := r.Context()

		if currentConn == 1 {
			// First connection: send msg-1 then immediately close connection to trigger client reconnect
			msgPayload := proto.Message{
				ID:        "msg-conn-1",
				SessionID: "sess-1",
				Role:      proto.Assistant,
			}
			rawMsg, _ := json.Marshal(pubsub.Event[proto.Message]{Type: pubsub.CreatedEvent, Payload: msgPayload})
			envData, _ := json.Marshal(pubsub.Payload{
				Type:    pubsub.PayloadTypeMessage,
				Payload: json.RawMessage(rawMsg),
			})
			_ = conn.Write(ctx, websocket.MessageText, envData)
			// Close connection abruptly
			_ = conn.Close(websocket.StatusGoingAway, "server restarting")
			return
		}

		// Second connection: send msg-2 and keep open
		msgPayload := proto.Message{
			ID:        "msg-conn-2",
			SessionID: "sess-1",
			Role:      proto.Assistant,
		}
		rawMsg, _ := json.Marshal(pubsub.Event[proto.Message]{Type: pubsub.CreatedEvent, Payload: msgPayload})
		envData, _ := json.Marshal(pubsub.Payload{
			Type:    pubsub.PayloadTypeMessage,
			Payload: json.RawMessage(rawMsg),
		})
		_ = conn.Write(ctx, websocket.MessageText, envData)

		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	c, err := client.NewClient(t.TempDir(), "tcp", u.Host)
	require.NoError(t, err)
	c.SetTransport("ws")

	ws := NewClientWorkspace(c, proto.Workspace{ID: "ws-1"})
	defer ws.Shutdown()

	received := make(chan tea.Msg, 20)
	go ws.runSubscription(func(msg tea.Msg) {
		received <- msg
	})

	// Wait for msg-conn-1
	var gotMsg1, gotMsg2 bool
	var gotDegraded, gotRecovered bool

	timeout := time.After(6 * time.Second)
	for !gotMsg2 {
		select {
		case msg := <-received:
			switch m := msg.(type) {
			case pubsub.Event[message.Message]:
				if m.Payload.ID == "msg-conn-1" {
					gotMsg1 = true
				} else if m.Payload.ID == "msg-conn-2" {
					gotMsg2 = true
				}
			case ConnectionEvent:
				if m.State == ConnectionDegraded {
					gotDegraded = true
				} else if m.State == ConnectionRecovered {
					gotRecovered = true
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for events. gotMsg1=%v, gotMsg2=%v, gotDegraded=%v, gotRecovered=%v",
				gotMsg1, gotMsg2, gotDegraded, gotRecovered)
		}
	}

	require.True(t, gotMsg1)
	require.True(t, gotDegraded)
	require.True(t, gotRecovered)
	require.True(t, gotMsg2)

	mu.Lock()
	require.GreaterOrEqual(t, connCount, 2)
	mu.Unlock()
}
