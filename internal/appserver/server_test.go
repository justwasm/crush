package appserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAppServer_InitializeAndPing(t *testing.T) {
	srv := NewServer(nil, nil, nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"test-client","version":"1.0"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"unknown/method"}` + "\n"

	in := strings.NewReader(input)
	var out bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := srv.Serve(ctx, in, &out)
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}

	scanner := bufio.NewScanner(&out)
	var responses []rpcResponse
	for scanner.Scan() {
		var resp rpcResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("Unmarshal error on line %q: %v", scanner.Text(), err)
		}
		responses = append(responses, resp)
	}

	if len(responses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(responses))
	}

	// Response 1: initialize
	if responses[0].ID != float64(1) {
		t.Errorf("resp 1 ID = %v, want 1", responses[0].ID)
	}
	res1, ok := responses[0].Result.(map[string]any)
	if !ok || res1["userAgent"] != "crush/1.0" {
		t.Errorf("resp 1 Result = %v, want userAgent 'crush/1.0'", responses[0].Result)
	}

	// Response 2: ping
	if responses[1].ID != float64(2) {
		t.Errorf("resp 2 ID = %v, want 2", responses[1].ID)
	}
	res2, ok := responses[1].Result.(map[string]any)
	if !ok || res2["pong"] != true {
		t.Errorf("resp 2 Result = %v, want pong true", responses[1].Result)
	}

	// Response 3: unknown method error
	if responses[2].ID != float64(3) {
		t.Errorf("resp 3 ID = %v, want 3", responses[2].ID)
	}
	if responses[2].Error == nil || responses[2].Error.Code != -32601 {
		t.Errorf("resp 3 Error = %v, want code -32601", responses[2].Error)
	}
}
