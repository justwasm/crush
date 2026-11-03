package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/coder/websocket"
)

func TestWebSocketEndpoint_HandshakeAndRPC(t *testing.T) {
	cfgStore, err := config.Load(context.Background(), t.TempDir(), t.TempDir(), false)
	if err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	srv := NewServer(cfgStore, "tcp", "127.0.0.1:0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket.Dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 1. Send initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"ws-test","version":"1.0"}}}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(initReq)); err != nil {
		t.Fatalf("conn.Write error: %v", err)
	}

	_, initRespBytes, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("conn.Read initialize response error: %v", err)
	}

	var initResp struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      uint64         `json:"id"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(initRespBytes, &initResp); err != nil {
		t.Fatalf("unmarshal initialize error: %v, body=%s", err, string(initRespBytes))
	}
	if initResp.ID != 1 {
		t.Errorf("expected ID 1, got %d", initResp.ID)
	}
	if initResp.Result["userAgent"] != "crush/1.0" {
		t.Errorf("expected userAgent crush/1.0, got %v", initResp.Result["userAgent"])
	}

	// 2. Send ping request
	pingReq := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(pingReq)); err != nil {
		t.Fatalf("conn.Write ping error: %v", err)
	}

	_, pingRespBytes, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("conn.Read ping response error: %v", err)
	}

	var pingResp struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      uint64         `json:"id"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(pingRespBytes, &pingResp); err != nil {
		t.Fatalf("unmarshal ping error: %v, body=%s", err, string(pingRespBytes))
	}
	if pingResp.ID != 2 {
		t.Errorf("expected ID 2, got %d", pingResp.ID)
	}
	if pingResp.Result["pong"] != true {
		t.Errorf("expected pong true, got %v", pingResp.Result["pong"])
	}

	// 3. Close connection gracefully
	if err := conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
		t.Errorf("conn.Close error: %v", err)
	}
}
