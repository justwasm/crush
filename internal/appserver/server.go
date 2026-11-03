package appserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/backend"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/google/uuid"
)

// Server serves the Crush app-server protocol (JSON-RPC 2.0 over NDJSON).
type Server struct {
	backend *backend.Backend
	cfg     *config.ConfigStore
	logger  *slog.Logger

	mu          sync.Mutex
	workspaces  map[string]*threadState // threadID (sessionID) -> threadState
	activeRunID string

	outMu sync.Mutex
	out   io.Writer
}

type threadState struct {
	workspaceID string
	sessionID   string
	cancel      context.CancelFunc

	mu                sync.Mutex
	lastTextByMsg     map[string]int
	lastThinkingByMsg map[string]int
	sentToolStarts    map[string]bool
	sentToolEnds      map[string]bool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// NewServer creates a new AppServer instance.
func NewServer(cfg *config.ConfigStore, b *backend.Backend, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		backend:    b,
		cfg:        cfg,
		logger:     logger,
		workspaces: make(map[string]*threadState),
	}
}

// Serve reads JSON-RPC requests from in and writes responses/notifications to out.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	s.out = out

	scanner := bufio.NewScanner(in)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 16*1024*1024) // Allow up to 16MB per NDJSON line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = s.replyError(nil, -32700, "Parse error: "+err.Error())
			continue
		}

		s.handleRequest(ctx, req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(ctx context.Context, req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "thread/start":
		s.handleThreadStart(ctx, req)
	case "turn/start":
		s.handleTurnStart(ctx, req)
	case "turn/cancel", "turn/interrupt":
		s.handleTurnCancel(ctx, req)
	case "model/list":
		s.handleModelList(req)
	case "ping":
		_ = s.reply(req.ID, map[string]any{"pong": true})
	default:
		_ = s.replyError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req rpcRequest) {
	home, _ := os.UserHomeDir()
	res := map[string]any{
		"userAgent": "crush/1.0",
		"codexHome": filepath.Join(home, ".config", "crush"),
		"capabilities": map[string]any{
			"threads": true,
			"turns":   true,
			"tools":   true,
		},
	}
	_ = s.reply(req.ID, res)
}

func (s *Server) handleThreadStart(ctx context.Context, req rpcRequest) {
	var params struct {
		CWD            string `json:"cwd"`
		Model          string `json:"model"`
		ApprovalPolicy string `json:"approvalPolicy"`
	}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	cwd := params.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	permMode := proto.WorkspacePermissionModeNormal
	if params.ApprovalPolicy == "never" || params.ApprovalPolicy == "" {
		permMode = proto.WorkspacePermissionModeYolo
	}

	clientID := uuid.New().String()
	ws, _, err := s.backend.CreateWorkspace(proto.Workspace{
		Path:           cwd,
		ClientID:       clientID,
		PermissionMode: permMode,
	})
	if err != nil {
		_ = s.replyError(req.ID, -32603, "failed to create workspace: "+err.Error())
		return
	}

	sess, err := s.backend.CreateSession(ctx, ws.ID, "Pigasus Session")
	if err != nil {
		_ = s.replyError(req.ID, -32603, "failed to create session: "+err.Error())
		return
	}

	evCtx, cancel := context.WithCancel(ctx)
	events, err := s.backend.SubscribeEvents(evCtx, ws.ID)
	if err != nil {
		cancel()
		_ = s.replyError(req.ID, -32603, "failed to subscribe events: "+err.Error())
		return
	}

	th := &threadState{
		workspaceID:       ws.ID,
		sessionID:         sess.ID,
		cancel:            cancel,
		lastTextByMsg:     make(map[string]int),
		lastThinkingByMsg: make(map[string]int),
		sentToolStarts:    make(map[string]bool),
		sentToolEnds:      make(map[string]bool),
	}

	s.mu.Lock()
	s.workspaces[sess.ID] = th
	s.mu.Unlock()

	go s.listenEvents(evCtx, th, events)

	activeModel := params.Model
	if activeModel == "" {
		activeModel = "crush-default"
	}

	res := map[string]any{
		"model":           activeModel,
		"modelProvider":   "crush",
		"reasoningEffort": "medium",
		"thread": map[string]any{
			"id":              sess.ID,
			"model":           activeModel,
			"modelProvider":   "crush",
			"reasoningEffort": "medium",
		},
	}
	_ = s.reply(req.ID, res)
}

func (s *Server) handleTurnStart(ctx context.Context, req rpcRequest) {
	var params struct {
		ThreadID string `json:"threadId"`
		Input    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"input"`
		Model  string `json:"model"`
		Effort string `json:"effort"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		_ = s.replyError(req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	s.mu.Lock()
	th, ok := s.workspaces[params.ThreadID]
	s.mu.Unlock()
	if !ok {
		_ = s.replyError(req.ID, -32602, "Thread not found: "+params.ThreadID)
		return
	}

	var prompt string
	for _, item := range params.Input {
		if item.Type == "text" || item.Type == "" {
			prompt += item.Text
		}
	}

	runID := fmt.Sprintf("run_%d", time.Now().UnixNano())

	th.mu.Lock()
	th.lastTextByMsg = make(map[string]int)
	th.lastThinkingByMsg = make(map[string]int)
	th.sentToolStarts = make(map[string]bool)
	th.sentToolEnds = make(map[string]bool)
	th.mu.Unlock()

	s.mu.Lock()
	s.activeRunID = runID
	s.mu.Unlock()

	err := s.backend.SendMessage(th.workspaceID, proto.AgentMessage{
		SessionID: th.sessionID,
		RunID:     runID,
		Prompt:    prompt,
	})
	if err != nil {
		_ = s.replyError(req.ID, -32603, "failed to send message: "+err.Error())
		return
	}

	_ = s.reply(req.ID, map[string]any{
		"turnId": runID,
		"turn": map[string]any{
			"id":     runID,
			"status": "in_progress",
		},
	})
}

func (s *Server) handleTurnCancel(ctx context.Context, req rpcRequest) {
	s.mu.Lock()
	for _, th := range s.workspaces {
		_ = s.backend.CancelSession(th.workspaceID, th.sessionID)
	}
	s.mu.Unlock()

	_ = s.reply(req.ID, map[string]any{
		"status": "cancelled",
	})
}

func (s *Server) handleModelList(req rpcRequest) {
	type codexReasoningEffort struct {
		ReasoningEffort string `json:"reasoningEffort"`
	}
	type modelInfo struct {
		ID                        string                 `json:"id"`
		DisplayName               string                 `json:"displayName"`
		Description               string                 `json:"description,omitempty"`
		SupportedReasoningEfforts []codexReasoningEffort `json:"supportedReasoningEfforts,omitempty"`
		DefaultReasoningEffort    string                 `json:"defaultReasoningEffort,omitempty"`
		IsDefault                 bool                   `json:"isDefault,omitempty"`
	}

	models := []modelInfo{
		{
			ID:          "default",
			DisplayName: "Crush Default Model",
			Description: "Default configured model for Crush",
			IsDefault:   true,
		},
	}

	if s.cfg != nil {
		for id, p := range s.cfg.Config().Providers.Seq2() {
			if p.Disable {
				continue
			}
			for _, m := range p.Models {
				disp := m.Name
				if disp == "" {
					disp = m.ID
				}
				models = append(models, modelInfo{
					ID:          m.ID,
					DisplayName: fmt.Sprintf("%s (%s)", disp, id),
				})
			}
		}
	}

	_ = s.reply(req.ID, map[string]any{
		"data":   models,
		"models": models,
	})
}

func (s *Server) listenEvents(ctx context.Context, th *threadState, events <-chan pubsub.Event[tea.Msg]) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			s.processEvent(th, ev)
		}
	}
}

func (s *Server) processEvent(th *threadState, ev pubsub.Event[tea.Msg]) {
	switch e := ev.Payload.(type) {
	case pubsub.Event[message.Message]:
		s.handleMessage(th, e.Payload)
	case message.Message:
		s.handleMessage(th, e)

	case pubsub.Event[notify.RunComplete]:
		s.handleRunComplete(th, e.Payload.RunID, e.Payload.Error, e.Payload.Cancelled)
	case notify.RunComplete:
		s.handleRunComplete(th, e.RunID, e.Error, e.Cancelled)

	case pubsub.Event[proto.RunComplete]:
		s.handleRunComplete(th, e.Payload.RunID, e.Payload.Error, e.Payload.Cancelled)
	case proto.RunComplete:
		s.handleRunComplete(th, e.RunID, e.Error, e.Cancelled)

	case pubsub.Event[permission.PermissionRequest]:
		s.handlePermissionRequest(th, e.Payload)
	case permission.PermissionRequest:
		s.handlePermissionRequest(th, e)
	}
}

func (s *Server) handleMessage(th *threadState, msg message.Message) {
	if msg.SessionID != th.sessionID || msg.Role != message.Assistant {
		return
	}
	th.mu.Lock()
	defer th.mu.Unlock()

	for _, part := range msg.Parts {
		switch p := part.(type) {
		case message.ReasoningContent:
			thinking := p.Thinking
			prev := th.lastThinkingByMsg[msg.ID]
			if len(thinking) > prev {
				delta := thinking[prev:]
				th.lastThinkingByMsg[msg.ID] = len(thinking)
				_ = s.notify("item/reasoning/textDelta", map[string]any{"delta": delta})
			}

		case message.TextContent:
			text := p.Text
			prev := th.lastTextByMsg[msg.ID]
			if len(text) > prev {
				delta := text[prev:]
				th.lastTextByMsg[msg.ID] = len(text)
				_ = s.notify("item/agentMessage/delta", map[string]any{"delta": delta})
			}

		case message.ToolCall:
			if !th.sentToolStarts[p.ID] {
				th.sentToolStarts[p.ID] = true
				_ = s.notify("item/started", map[string]any{
					"item": map[string]any{
						"id":      p.ID,
						"type":    "commandExecution",
						"command": p.Name + " " + string(p.Input),
					},
				})
			}

		case message.ToolResult:
			if !th.sentToolEnds[p.ToolCallID] {
				th.sentToolEnds[p.ToolCallID] = true
				exitCode := 0
				if p.IsError {
					exitCode = 1
				}
				_ = s.notify("item/completed", map[string]any{
					"item": map[string]any{
						"id":       p.ToolCallID,
						"type":     "commandExecution",
						"output":   p.Content,
						"exitCode": exitCode,
					},
				})
			}
		}
	}
}

func (s *Server) handleRunComplete(th *threadState, runID, errMsg string, cancelled bool) {
	s.mu.Lock()
	active := s.activeRunID
	s.mu.Unlock()

	if runID != "" && active != "" && runID != active {
		return
	}

	status := "completed"
	if cancelled {
		status = "cancelled"
	} else if errMsg != "" {
		status = "failed"
	}

	turnMap := map[string]any{
		"id":     runID,
		"status": status,
	}
	if errMsg != "" {
		turnMap["error"] = map[string]any{
			"message": errMsg,
		}
	}

	_ = s.notify("turn/completed", map[string]any{
		"threadId": th.sessionID,
		"turnId":   runID,
		"turn":     turnMap,
	})
}

func (s *Server) handlePermissionRequest(th *threadState, e permission.PermissionRequest) {
	// Automatically grant permissions in non-interactive/remote mode
	_, _ = s.backend.GrantPermission(th.workspaceID, proto.PermissionGrant{
		Action: proto.PermissionAllow,
		Permission: proto.PermissionRequest{
			ID:         e.ID,
			SessionID:  e.SessionID,
			ToolCallID: e.ToolCallID,
		},
	})
}

func (s *Server) reply(id any, result any) error {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.writeLine(resp)
}

func (s *Server) replyError(id any, code int, message string) error {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	}
	return s.writeLine(resp)
}

func (s *Server) notify(method string, params any) error {
	notif := rpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return s.writeLine(notif)
}

func (s *Server) writeLine(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	_, err = s.out.Write(append(data, '\n'))
	return err
}
