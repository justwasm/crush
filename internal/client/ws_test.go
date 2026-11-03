package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/server"
	"github.com/coder/websocket"
)

func TestWSClient_EndToEnd(t *testing.T) {
	cfgStore, err := config.Load(context.Background(), t.TempDir(), t.TempDir(), false)
	if err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	srv := server.NewServer(cfgStore, "tcp", "127.0.0.1:0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/ws"
	wsClient, err := DialWS(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("DialWS error: %v", err)
	}
	defer wsClient.Close()

	// 1. Initialize
	initRes, err := wsClient.Initialize(ctx, "test-client", "1.0.0")
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}
	if initRes["userAgent"] != "crush/1.0" {
		t.Errorf("expected userAgent crush/1.0, got %v", initRes["userAgent"])
	}

	// 2. Ping
	if err := wsClient.Ping(ctx); err != nil {
		t.Fatalf("Ping error: %v", err)
	}

	// 3. ListModels
	models, err := wsClient.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}
	if len(models) == 0 {
		t.Errorf("expected at least 1 model, got 0")
	}

	// 4. ThreadStart
	th, err := wsClient.ThreadStart(ctx, t.TempDir(), "default", "never")
	if err != nil {
		t.Fatalf("ThreadStart error: %v", err)
	}
	if th.ID == "" {
		t.Errorf("expected non-empty thread ID")
	}
	if wsClient.ThreadID() != th.ID {
		t.Errorf("expected ThreadID() %s, got %s", th.ID, wsClient.ThreadID())
	}

	// 5. TurnStart (round-trip test over WebSocket to backend)
	_, _ = wsClient.TurnStart(ctx, th.ID, "hello from e2e", "default", "low")

	// 6. TurnCancel
	if err := wsClient.TurnCancel(ctx); err != nil {
		t.Fatalf("TurnCancel error: %v", err)
	}
}

func TestWSClient_ClientDialWS(t *testing.T) {
	cfgStore, err := config.Load(context.Background(), t.TempDir(), t.TempDir(), false)
	if err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	srv := server.NewServer(cfgStore, "tcp", "127.0.0.1:0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse host:port from test server
	addr := strings.TrimPrefix(ts.URL, "http://")
	c, err := NewClient(t.TempDir(), "tcp", addr)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	// Test c.DialWS
	wsClient, err := c.DialWS(ctx)
	if err != nil {
		t.Fatalf("c.DialWS error: %v", err)
	}
	defer wsClient.Close()

	if err := wsClient.Ping(ctx); err != nil {
		t.Fatalf("wsClient.Ping error: %v", err)
	}
}

func TestWSClient_StreamingAndApproval(t *testing.T) {
	// Mock server that emits reasoning, text delta, tool call, turn completion, and approval request
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusInternalError, "done")

		ctx := r.Context()

		// Read client initialize
		var req struct {
			ID     uint64 `json:"id"`
			Method string `json:"method"`
		}
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		_ = json.Unmarshal(data, &req)

		// Reply to initialize
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","id":1,"result":{"userAgent":"mock/1.0"}}`))

		// Push notification: reasoning delta
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","method":"item/reasoning/textDelta","params":{"delta":"thinking..."}}`))

		// Push notification: agent message delta
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","method":"item/agentMessage/delta","params":{"delta":"hello world"}}`))

		// Push notification: tool started
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","method":"item/started","params":{"item":{"id":"tool_1","type":"commandExecution","command":"ls -la"}}}`))

		// Push notification: tool completed
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","method":"item/completed","params":{"item":{"id":"tool_1","type":"commandExecution","output":"file1\nfile2","exitCode":0}}}`))

		// Push server request: approval
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","id":999,"method":"item/commandExecution/requestApproval","params":{"command":"rm -rf /tmp/test"}}`))

		// Read client reply to approval
		_, replyData, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var approvalReply struct {
			ID     uint64         `json:"id"`
			Result map[string]any `json:"result"`
		}
		_ = json.Unmarshal(replyData, &approvalReply)
		if approvalReply.ID != 999 || approvalReply.Result["decision"] != "custom_decision" {
			t.Errorf("unexpected approval reply: %s", string(replyData))
		}

		// Push notification: turn completed
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","method":"turn/completed","params":{"threadId":"t1","turnId":"turn_1","turn":{"id":"turn_1","status":"completed"}}}`))

		// Wait briefly before clean exit
		time.Sleep(50 * time.Millisecond)
	})

	ts := httptest.NewServer(serverHandler)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := DialWS(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("DialWS error: %v", err)
	}
	defer client.Close()

	client.SetApprovalHandler(func(req WSApprovalRequest) string {
		if req.Method == "item/commandExecution/requestApproval" {
			return "custom_decision"
		}
		return "approved"
	})

	// Handshake
	_, err = client.Initialize(ctx, "mock-client", "1.0")
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}

	// Verify events received on Events() channel
	events := make([]WSEvent, 0)
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case ev, ok := <-client.Events():
			if !ok {
				break loop
			}
			events = append(events, ev)
			if ev.Kind == WSEventTurnCompleted {
				break loop
			}
		case <-deadline:
			t.Fatalf("timed out waiting for events, got %d", len(events))
		}
	}

	if len(events) < 5 {
		t.Fatalf("expected at least 5 events, got %d", len(events))
	}

	if events[0].Kind != WSEventReasoningDelta || events[0].Delta != "thinking..." {
		t.Errorf("event 0 want reasoning delta 'thinking...', got %+v", events[0])
	}
	if events[1].Kind != WSEventTextDelta || events[1].Delta != "hello world" {
		t.Errorf("event 1 want text delta 'hello world', got %+v", events[1])
	}
	if events[2].Kind != WSEventToolStarted || events[2].ToolID != "tool_1" {
		t.Errorf("event 2 want tool start 'tool_1', got %+v", events[2])
	}
	if events[3].Kind != WSEventToolCompleted || events[3].ToolID != "tool_1" || events[3].Output != "file1\nfile2" {
		t.Errorf("event 3 want tool complete 'tool_1', got %+v", events[3])
	}
	if events[4].Kind != WSEventTurnCompleted || events[4].TurnID != "turn_1" || events[4].Status != "completed" {
		t.Errorf("event 4 want turn complete, got %+v", events[4])
	}
}
