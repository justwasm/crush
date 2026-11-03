package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWebSocketEvents_StreamAndFilter(t *testing.T) {
	h := newE2EHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client1ID := uuid.New().String()
	client2ID := uuid.New().String()

	// Client 1 dials /v1/workspaces/{id}/events via WebSocket
	wsURL1 := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/" + h.workspace.ID + "/events?client_id=" + client1ID
	conn1, _, err := websocket.Dial(ctx, wsURL1, nil)
	require.NoError(t, err)
	defer conn1.Close(websocket.StatusNormalClosure, "")

	// Client 2 dials /v1/workspaces/{id}/ws via WebSocket
	wsURL2 := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/" + h.workspace.ID + "/ws?client_id=" + client2ID
	conn2, _, err := websocket.Dial(ctx, wsURL2, nil)
	require.NoError(t, err)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Client 1 sets its viewed session to "sess-1" via control message
	setSessMsg1, _ := json.Marshal(map[string]string{
		"type":       "current_session",
		"session_id": "sess-1",
	})
	require.NoError(t, conn1.Write(ctx, websocket.MessageText, setSessMsg1))

	// Client 2 sets its viewed session to "sess-2" via control message
	setSessMsg2, _ := json.Marshal(map[string]string{
		"type":       "current_session",
		"session_id": "sess-2",
	})
	require.NoError(t, conn2.Write(ctx, websocket.MessageText, setSessMsg2))

	// Allow backend to process the session attachment
	time.Sleep(50 * time.Millisecond)

	// Publish a workspace-wide message event
	msgEvent := message.Message{
		ID:        "msg-1",
		SessionID: "sess-common",
		Role:      message.User,
		Parts:     []message.ContentPart{message.TextContent{Text: "Hello WS"}},
	}
	h.app.SendEvent(pubsub.Event[message.Message]{
		Type:    pubsub.CreatedEvent,
		Payload: msgEvent,
	})

	// Both client 1 and client 2 should receive the workspace-wide message event
	readUntil := func(conn *websocket.Conn, targetType pubsub.PayloadType) pubsub.Payload {
		for {
			_, data, err := conn.Read(ctx)
			require.NoError(t, err)
			var p pubsub.Payload
			require.NoError(t, json.Unmarshal(data, &p))
			if p.Type == targetType {
				return p
			}
		}
	}

	p1 := readUntil(conn1, pubsub.PayloadTypeMessage)
	require.Equal(t, pubsub.PayloadTypeMessage, p1.Type)

	p2 := readUntil(conn2, pubsub.PayloadTypeMessage)
	require.Equal(t, pubsub.PayloadTypeMessage, p2.Type)

	// Publish a session-scoped permission request for sess-1
	permReq := permission.PermissionRequest{
		ID:        "perm-1",
		SessionID: "sess-1",
		Action:    "read_file",
		Path:      "/tmp/test",
	}
	h.app.SendEvent(pubsub.Event[permission.PermissionRequest]{
		Type:    pubsub.CreatedEvent,
		Payload: permReq,
	})

	// Client 1 is viewing sess-1 so it MUST receive the permission request
	p1Perm := readUntil(conn1, pubsub.PayloadTypePermissionRequest)
	require.Equal(t, pubsub.PayloadTypePermissionRequest, p1Perm.Type)

	// Client 2 is NOT viewing sess-1, so it must not receive this event
	readTimeoutCtx, readCancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer readCancel()
	_, _, err = conn2.Read(readTimeoutCtx)
	require.Error(t, err, "Client 2 should not have received session-scoped event for sess-1")
}

func TestWebSocketEvents_PingPong(t *testing.T) {
	h := newE2EHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientID := uuid.New().String()
	wsURL := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/" + h.workspace.ID + "/events?client_id=" + clientID
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	require.NoError(t, conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)))

	_, data, err := conn.Read(ctx)
	require.NoError(t, err)

	var resp struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(data, &resp))
	require.Equal(t, "pong", resp.Type)
}

func TestWebSocketEvents_NotFound(t *testing.T) {
	h := newE2EHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientID := uuid.New().String()
	wsURL := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/nonexistent-ws/events?client_id=" + clientID
	_, rsp, err := websocket.Dial(ctx, wsURL, nil)
	require.Error(t, err)
	if rsp != nil {
		require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestWebSocketEvents_InvalidClientID(t *testing.T) {
	h := newE2EHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Missing client_id
	wsURL := "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/" + h.workspace.ID + "/events"
	_, rsp, err := websocket.Dial(ctx, wsURL, nil)
	require.Error(t, err)
	if rsp != nil {
		require.Equal(t, http.StatusBadRequest, rsp.StatusCode)
	}

	// Invalid UUID
	wsURL = "ws" + strings.TrimPrefix(h.httpSrv.URL, "http") + "/v1/workspaces/" + h.workspace.ID + "/events?client_id=not-a-uuid"
	_, rsp, err = websocket.Dial(ctx, wsURL, nil)
	require.Error(t, err)
	if rsp != nil {
		require.Equal(t, http.StatusBadRequest, rsp.StatusCode)
	}
}
