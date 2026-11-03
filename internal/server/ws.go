package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/charmbracelet/crush/internal/appserver"
	"github.com/coder/websocket"
)

// handleWebSocket upgrades incoming HTTP connections to a full-duplex WebSocket
// and serves the Crush JSON-RPC 2.0 app-server protocol over it.
func (c *controllerV1) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	opts := &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from IDEs, local harnesses, and web dashboards
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		c.server.logError(r, "Failed to accept websocket connection", "error", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "closing")

	ctx := r.Context()
	logger := c.server.logger
	if logger == nil {
		logger = slog.Default()
	}

	rw := newWSReadWriter(ctx, conn)
	srv := appserver.NewServer(c.server.cfg, c.backend, logger)

	if err := srv.Serve(ctx, rw, rw); err != nil {
		if !isNormalClose(err) {
			logger.Debug("WebSocket JSON-RPC session closed", "error", err)
		}
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func isNormalClose(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway
}

// wsReadWriter wraps a WebSocket connection to implement io.Reader and io.Writer
// with newline-delimited framing guarantees for NDJSON/JSON-RPC.
type wsReadWriter struct {
	ctx     context.Context
	conn    *websocket.Conn
	mu      sync.Mutex
	writeMu sync.Mutex
	readBuf bytes.Buffer
}

func newWSReadWriter(ctx context.Context, conn *websocket.Conn) *wsReadWriter {
	return &wsReadWriter{
		ctx:  ctx,
		conn: conn,
	}
}

func (rw *wsReadWriter) Read(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	for rw.readBuf.Len() == 0 {
		rw.mu.Unlock()
		msgType, data, err := rw.conn.Read(rw.ctx)
		rw.mu.Lock()
		if err != nil {
			return 0, err
		}
		if msgType != websocket.MessageText {
			continue
		}
		if len(data) == 0 {
			continue
		}
		rw.readBuf.Write(data)
		if !bytes.HasSuffix(data, []byte("\n")) {
			rw.readBuf.WriteByte('\n')
		}
	}

	return rw.readBuf.Read(p)
}

func (rw *wsReadWriter) Write(p []byte) (int, error) {
	rw.writeMu.Lock()
	defer rw.writeMu.Unlock()

	err := rw.conn.Write(rw.ctx, websocket.MessageText, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
