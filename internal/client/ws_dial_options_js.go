//go:build js

package client

import (
	"net/http"

	"github.com/coder/websocket"
)

func wsDialOptions(_ *http.Client) *websocket.DialOptions {
	return nil
}
