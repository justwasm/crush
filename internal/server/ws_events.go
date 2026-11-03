package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/coder/websocket"
)

func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// handleWorkspaceEventsWS handles workspace event streaming over WebSocket.
func (c *controllerV1) handleWorkspaceEventsWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	clientID, ok := c.requireClientID(w, r)
	if !ok {
		return
	}

	// Verify workspace exists and subscribe to events before upgrading the connection.
	// If the workspace does not exist, this returns 404 cleanly via HTTP status before WebSocket handshake.
	events, err := c.backend.SubscribeEvents(r.Context(), id)
	if err != nil {
		c.handleError(w, r, err)
		return
	}
	sessions, err := c.backend.SubscribeClientSessions(r.Context(), id)
	if err != nil {
		c.handleError(w, r, err)
		return
	}

	opts := &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		c.server.logError(r, "Failed to accept websocket events connection", "error", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	if err := c.backend.AttachClient(id, clientID); err != nil {
		c.server.logError(r, "Failed to attach client", "error", err)
		_ = conn.Close(websocket.StatusInternalError, "failed to attach client")
		return
	}
	defer c.backend.DetachClient(id, clientID)

	var writeMu sync.Mutex
	writeWSEvent := func(ev any) error {
		wrapped := wrapEvent(ev)
		if wrapped == nil {
			return nil
		}
		data, err := json.Marshal(wrapped)
		if err != nil {
			c.server.logError(r, "Failed to marshal websocket event", "error", err)
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.Write(ctx, websocket.MessageText, data)
	}

	// Full-duplex reader: monitors connection state and handles any client control messages
	go func() {
		for {
			msgType, data, err := conn.Read(ctx)
			if err != nil {
				cancel()
				return
			}
			if msgType == websocket.MessageText && len(data) > 0 {
				var control struct {
					Type      string `json:"type"`
					SessionID string `json:"session_id,omitempty"`
				}
				if err := json.Unmarshal(data, &control); err == nil {
					switch control.Type {
					case "ping":
						writeMu.Lock()
						_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"pong"}`))
						writeMu.Unlock()
					case "current_session":
						_ = c.backend.SetCurrentSession(ctx, id, clientID, control.SessionID)
					}
				}
			}
		}
	}()

	// Replay initial state (theme and client sessions)
	if st, err := c.backend.GetHostTheme(id); err == nil && st.Theme != nil {
		_ = writeWSEvent(pubsub.Event[proto.HostThemeState]{
			Type:    pubsub.UpdatedEvent,
			Payload: st,
		})
	}
	if clientSessions, err := c.backend.ClientSessions(id); err == nil {
		for _, s := range clientSessions {
			_ = writeWSEvent(pubsub.Event[proto.ClientSession]{
				Type:    pubsub.UpdatedEvent,
				Payload: s,
			})
		}
	}

	for {
		select {
		case <-ctx.Done():
			c.server.logDebug(r, "Stopping websocket event stream")
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case ev, ok := <-events:
			if !ok {
				_ = conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if !c.deliverToClient(ctx, id, clientID, ev.Payload) {
				continue
			}
			if err := writeWSEvent(ev.Payload); err != nil {
				return
			}
		case sev, ok := <-sessions:
			if !ok {
				sessions = nil
				continue
			}
			if err := writeWSEvent(sev); err != nil {
				return
			}
		}
	}
}
