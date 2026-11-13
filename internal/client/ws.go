package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
)

// WSEventKind represents the type of an event received from Crush app-server over WebSocket.
type WSEventKind string

const (
	WSEventReasoningDelta WSEventKind = "reasoning_delta"
	WSEventTextDelta      WSEventKind = "text_delta"
	WSEventToolStarted    WSEventKind = "tool_started"
	WSEventToolCompleted  WSEventKind = "tool_completed"
	WSEventTurnCompleted  WSEventKind = "turn_completed"
	WSEventError          WSEventKind = "error"
)

// WSEvent represents a streamed event from the Crush server.
type WSEvent struct {
	Kind     WSEventKind `json:"kind"`
	ThreadID string      `json:"thread_id,omitempty"`
	TurnID   string      `json:"turn_id,omitempty"`
	Delta    string      `json:"delta,omitempty"`
	ToolID   string      `json:"tool_id,omitempty"`
	Command  string      `json:"command,omitempty"`
	Output   string      `json:"output,omitempty"`
	ExitCode int         `json:"exit_code,omitempty"`
	Status   string      `json:"status,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// WSApprovalRequest represents an interactive approval request sent from the server.
type WSApprovalRequest struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// WSModelInfo represents model information returned by model/list.
type WSModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}

// WSThreadInfo represents thread information returned by thread/start.
type WSThreadInfo struct {
	ID            string `json:"id"`
	Model         string `json:"model"`
	ModelProvider string `json:"modelProvider"`
}

// WSDialOptions configures dialing a WebSocket connection.
type WSDialOptions struct {
	HTTPClient *http.Client
}

type wsRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type wsRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *uint64          `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *wsRPCError      `json:"error,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type wsRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WSClient is a full-duplex WebSocket JSON-RPC 2.0 client adapter for Crush app-server.
type WSClient struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc

	writeMu sync.Mutex

	mu              sync.Mutex
	reqSeq          uint64
	pendingReqs     map[uint64]chan wsRPCResponse
	eventsCh        chan WSEvent
	threadID        string
	activeTurn      string
	closed          bool
	approvalHandler func(req WSApprovalRequest) string
}

// DialWS dials the given WebSocket URL and returns an active WSClient.
func DialWS(ctx context.Context, u string, opts *WSDialOptions) (*WSClient, error) {
	var dialOpts *websocket.DialOptions
	if opts != nil && opts.HTTPClient != nil {
		dialOpts = wsDialOptions(opts.HTTPClient)
	}

	conn, _, err := websocket.Dial(ctx, u, dialOpts)
	if err != nil {
		return nil, fmt.Errorf("websocket dial %s: %w", u, err)
	}

	clientCtx, clientCancel := context.WithCancel(context.Background())
	client := &WSClient{
		conn:        conn,
		ctx:         clientCtx,
		cancel:      clientCancel,
		pendingReqs: make(map[uint64]chan wsRPCResponse),
		eventsCh:    make(chan WSEvent, 128),
	}

	go client.readLoop()
	return client, nil
}

// DialWS dials the /v1/ws WebSocket route on the server using the client's configured network and address.
func (c *Client) DialWS(ctx context.Context) (*WSClient, error) {
	switch c.network {
	case "tcp":
		wsURL := fmt.Sprintf("ws://%s/v1/ws", c.addr)
		return DialWS(ctx, wsURL, nil)
	case "unix", "npipe":
		tr := &http.Transport{
			DialContext: c.dialer,
		}
		httpClient := &http.Client{Transport: tr}
		return DialWS(ctx, "ws://crush.local/v1/ws", &WSDialOptions{HTTPClient: httpClient})
	default:
		return nil, fmt.Errorf("unsupported client network for websocket: %s", c.network)
	}
}

// Events returns the stream of events emitted by the server during turn execution.
func (c *WSClient) Events() <-chan WSEvent {
	return c.eventsCh
}

// SetApprovalHandler sets a custom handler for interactive approval requests.
func (c *WSClient) SetApprovalHandler(handler func(req WSApprovalRequest) string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.approvalHandler = handler
}

// ThreadID returns the currently active thread ID.
func (c *WSClient) ThreadID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.threadID
}

func (c *WSClient) nextID() uint64 {
	return atomic.AddUint64(&c.reqSeq, 1)
}

// Call performs a synchronous JSON-RPC 2.0 request and decodes the result.
func (c *WSClient) Call(ctx context.Context, method string, params any, result any) error {
	id := c.nextID()
	respCh := make(chan wsRPCResponse, 1)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("crush ws client is closed")
	}
	c.pendingReqs[id] = respCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingReqs, id)
		c.mu.Unlock()
	}()

	req := wsRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	err = c.conn.Write(ctx, websocket.MessageText, data)
	c.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("crush ws write request %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return errors.New("crush ws connection closed")
	case resp := <-respCh:
		if resp.Error != nil {
			return fmt.Errorf("crush jsonrpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		if result != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("crush unmarshal %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *WSClient) reply(id uint64, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	data, err := json.Marshal(resp)
	if err == nil {
		c.writeMu.Lock()
		_ = c.conn.Write(c.ctx, websocket.MessageText, data)
		c.writeMu.Unlock()
	}
}

// Initialize performs the handshake with Crush app-server.
func (c *WSClient) Initialize(ctx context.Context, name, version string) (map[string]any, error) {
	type clientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	type initParams struct {
		ClientInfo clientInfo `json:"clientInfo"`
	}
	var res map[string]any
	err := c.Call(ctx, "initialize", initParams{
		ClientInfo: clientInfo{Name: name, Version: version},
	}, &res)
	return res, err
}

// ThreadStart initializes a new thread session on the server.
func (c *WSClient) ThreadStart(ctx context.Context, cwd, model, approvalPolicy string) (*WSThreadInfo, error) {
	type threadParams struct {
		CWD            string `json:"cwd,omitempty"`
		Model          string `json:"model,omitempty"`
		ApprovalPolicy string `json:"approvalPolicy,omitempty"`
	}
	var threadResp struct {
		Model         string `json:"model"`
		ModelProvider string `json:"modelProvider"`
		Thread        struct {
			ID            string `json:"id"`
			Model         string `json:"model"`
			ModelProvider string `json:"modelProvider"`
		} `json:"thread"`
	}

	if err := c.Call(ctx, "thread/start", threadParams{
		CWD:            cwd,
		Model:          model,
		ApprovalPolicy: approvalPolicy,
	}, &threadResp); err != nil {
		return nil, err
	}

	id := threadResp.Thread.ID
	c.mu.Lock()
	c.threadID = id
	c.mu.Unlock()

	return &WSThreadInfo{
		ID:            id,
		Model:         threadResp.Model,
		ModelProvider: threadResp.ModelProvider,
	}, nil
}

// TurnStart begins a new conversational turn and returns the generated turn ID.
func (c *WSClient) TurnStart(ctx context.Context, threadID, prompt, model, effort string) (string, error) {
	type inputItem struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type turnParams struct {
		ThreadID string      `json:"threadId"`
		Input    []inputItem `json:"input"`
		Model    string      `json:"model,omitempty"`
		Effort   string      `json:"effort,omitempty"`
	}

	var resp struct {
		TurnID string `json:"turnId"`
	}

	err := c.Call(ctx, "turn/start", turnParams{
		ThreadID: threadID,
		Input:    []inputItem{{Type: "text", Text: prompt}},
		Model:    model,
		Effort:   effort,
	}, &resp)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.activeTurn = resp.TurnID
	c.mu.Unlock()
	return resp.TurnID, nil
}

// TurnCancel requests interruption of the active turn.
func (c *WSClient) TurnCancel(ctx context.Context) error {
	var resp map[string]any
	return c.Call(ctx, "turn/cancel", map[string]any{}, &resp)
}

// ListModels retrieves the list of configured models from the server.
func (c *WSClient) ListModels(ctx context.Context) ([]WSModelInfo, error) {
	var resp struct {
		Models []WSModelInfo `json:"models"`
		Data   []WSModelInfo `json:"data"`
	}
	if err := c.Call(ctx, "model/list", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Models) > 0 {
		return resp.Models, nil
	}
	return resp.Data, nil
}

// Ping sends a ping and verifies server connectivity.
func (c *WSClient) Ping(ctx context.Context) error {
	var resp struct {
		Pong bool `json:"pong"`
	}
	return c.Call(ctx, "ping", map[string]any{}, &resp)
}

// Close terminates the WebSocket connection and background reader.
func (c *WSClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.cancel()
	c.mu.Unlock()

	return c.conn.Close(websocket.StatusNormalClosure, "client closing")
}

func (c *WSClient) readLoop() {
	defer func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		for id, ch := range c.pendingReqs {
			ch <- wsRPCResponse{Error: &wsRPCError{Message: "connection closed"}}
			delete(c.pendingReqs, id)
		}
		close(c.eventsCh)
		c.mu.Unlock()
	}()

	for {
		msgType, data, err := c.conn.Read(c.ctx)
		if err != nil {
			return
		}
		if msgType != websocket.MessageText || len(data) == 0 {
			continue
		}

		var resp wsRPCResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}

		// 1. Response to client request
		if resp.ID != nil && resp.Method == "" {
			c.mu.Lock()
			ch, ok := c.pendingReqs[*resp.ID]
			c.mu.Unlock()
			if ok {
				ch <- resp
			}
			continue
		}

		// 2. Server-to-Client Request (Approval)
		if resp.ID != nil && resp.Method != "" {
			go c.handleServerRequest(*resp.ID, resp.Method, resp.Params)
			continue
		}

		// 3. Notification from Server
		if resp.Method != "" {
			c.handleNotification(resp.Method, resp.Params)
		}
	}
}

func (c *WSClient) handleServerRequest(id uint64, method string, params json.RawMessage) {
	c.mu.Lock()
	handler := c.approvalHandler
	c.mu.Unlock()

	decision := "approved"
	if handler != nil {
		decision = handler(WSApprovalRequest{
			ID:     id,
			Method: method,
			Params: params,
		})
	}
	c.reply(id, map[string]any{"decision": decision})
}

func (c *WSClient) handleNotification(method string, paramsRaw json.RawMessage) {
	switch method {
	case "item/reasoning/textDelta":
		var p struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(paramsRaw, &p); err == nil && p.Delta != "" {
			c.sendEvent(WSEvent{
				Kind:  WSEventReasoningDelta,
				Delta: p.Delta,
			})
		}

	case "item/agentMessage/delta":
		var p struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(paramsRaw, &p); err == nil && p.Delta != "" {
			c.sendEvent(WSEvent{
				Kind:  WSEventTextDelta,
				Delta: p.Delta,
			})
		}

	case "item/started":
		var p struct {
			Item struct {
				ID      string `json:"id"`
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"item"`
		}
		if err := json.Unmarshal(paramsRaw, &p); err == nil {
			c.sendEvent(WSEvent{
				Kind:    WSEventToolStarted,
				ToolID:  p.Item.ID,
				Command: p.Item.Command,
			})
		}

	case "item/completed":
		var p struct {
			Item struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				ExitCode int    `json:"exitCode"`
				Output   string `json:"output"`
			} `json:"item"`
		}
		if err := json.Unmarshal(paramsRaw, &p); err == nil {
			c.sendEvent(WSEvent{
				Kind:     WSEventToolCompleted,
				ToolID:   p.Item.ID,
				Output:   p.Item.Output,
				ExitCode: p.Item.ExitCode,
			})
		}

	case "turn/completed":
		var p struct {
			ThreadID string `json:"threadId"`
			TurnID   string `json:"turnId"`
			Turn     struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error,omitempty"`
			} `json:"turn"`
		}
		_ = json.Unmarshal(paramsRaw, &p)
		errMsg := ""
		if p.Turn.Error != nil && p.Turn.Error.Message != "" {
			errMsg = p.Turn.Error.Message
		}
		turnID := p.Turn.ID
		if turnID == "" {
			turnID = p.TurnID
		}
		c.sendEvent(WSEvent{
			Kind:     WSEventTurnCompleted,
			ThreadID: p.ThreadID,
			TurnID:   turnID,
			Status:   p.Turn.Status,
			Error:    errMsg,
		})

	case "error":
		var p struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(paramsRaw, &p); err == nil {
			c.sendEvent(WSEvent{
				Kind:  WSEventError,
				Error: p.Message,
			})
		}
	}
}

func (c *WSClient) sendEvent(ev WSEvent) {
	select {
	case c.eventsCh <- ev:
	default:
		// Do not block reader if events buffer is temporarily full
	}
}
