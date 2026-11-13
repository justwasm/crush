//go:build !js

package client

import (
	"net/http"

	"github.com/coder/websocket"
)

func wsDialOptions(client *http.Client) *websocket.DialOptions {
	if client == nil {
		return nil
	}
	return &websocket.DialOptions{
		HTTPClient: client,
	}
}
