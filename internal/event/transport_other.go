//go:build !(js && wasm)

package event

import (
	"net/http"
	"time"
)

func newNoCORSClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
