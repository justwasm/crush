package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/coder/websocket"
)

// SubscribeEventsWS subscribes to workspace events over a WebSocket connection.
func (c *Client) SubscribeEventsWS(ctx context.Context, id string) (<-chan any, error) {
	var wsURL string
	switch c.network {
	case "tcp":
		wsURL = fmt.Sprintf("ws://%s/v1/workspaces/%s/events?client_id=%s",
			c.addr, url.PathEscape(id), url.QueryEscape(c.clientID))
	case "unix", "npipe":
		wsURL = fmt.Sprintf("ws://%s/v1/workspaces/%s/events?client_id=%s",
			DummyHost, url.PathEscape(id), url.QueryEscape(c.clientID))
	default:
		return nil, fmt.Errorf("unsupported client network for websocket: %s", c.network)
	}

	var dialOpts *websocket.DialOptions
	if c.network == "unix" || c.network == "npipe" {
		tr := &http.Transport{DialContext: c.dialer}
		dialOpts = &websocket.DialOptions{
			HTTPClient: &http.Client{Transport: tr},
		}
	} else if c.h != nil {
		tr := c.h.Transport
		dialOpts = &websocket.DialOptions{
			HTTPClient: &http.Client{Transport: tr},
		}
	}

	conn, rsp, err := websocket.Dial(ctx, wsURL, dialOpts)
	if err != nil {
		if rsp != nil {
			if statusErr := checkStatus(rsp); statusErr != nil {
				return nil, fmt.Errorf("failed to subscribe to events: %w", statusErr)
			}
		}
		return nil, fmt.Errorf("failed to subscribe to events via websocket: %w", err)
	}

	events := make(chan any, 100)
	go func() {
		defer conn.Close(websocket.StatusNormalClosure, "")
		defer close(events)

		for {
			msgType, data, err := conn.Read(ctx)
			if err != nil {
				if ctx.Err() == nil && !isNormalWSClose(err) {
					slog.Warn("WebSocket events stream ended", "error", err)
				}
				return
			}
			if msgType != websocket.MessageText {
				continue
			}
			data = bytes.TrimSpace(data)
			if len(data) == 0 {
				continue
			}
			if !parseAndDispatchEvent(ctx, events, data) {
				return
			}
		}
	}()

	return events, nil
}

func isNormalWSClose(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway
}

// parseAndDispatchEvent unmarshals an event envelope from JSON data and delivers
// the typed event onto the events channel. It returns false if the context was cancelled.
func parseAndDispatchEvent(ctx context.Context, events chan any, data []byte) bool {
	var p pubsub.Payload
	if err := json.Unmarshal(data, &p); err != nil {
		slog.Error("Unmarshaling event envelope", "error", err)
		return true
	}

	switch p.Type {
	case pubsub.PayloadTypeLSPEvent:
		var e pubsub.Event[proto.LSPEvent]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeMCPEvent:
		var e pubsub.Event[proto.MCPEvent]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypePermissionRequest:
		var e pubsub.Event[proto.PermissionRequest]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypePermissionNotification:
		var e pubsub.Event[proto.PermissionNotification]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeQuestionRequest:
		var e pubsub.Event[proto.QuestionRequest]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeQuestionNotification:
		var e pubsub.Event[proto.QuestionNotification]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeMessage:
		var e pubsub.Event[proto.Message]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeSession:
		var e pubsub.Event[proto.Session]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeFile:
		var e pubsub.Event[proto.File]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeAgentEvent:
		var e pubsub.Event[proto.AgentEvent]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeConfigChanged:
		var e pubsub.Event[proto.ConfigChanged]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeSkillsEvent:
		var e pubsub.Event[proto.SkillsEvent]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeRunComplete:
		var e pubsub.Event[proto.RunComplete]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeUpdateAvailable:
		var e pubsub.Event[proto.UpdateAvailable]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeHostTheme:
		var e pubsub.Event[proto.HostThemeState]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	case pubsub.PayloadTypeClientSession:
		var e pubsub.Event[proto.ClientSession]
		_ = json.Unmarshal(p.Payload, &e)
		return sendEvent(ctx, events, e)
	default:
		slog.Warn("Unknown event type", "type", p.Type)
		return true
	}
}
