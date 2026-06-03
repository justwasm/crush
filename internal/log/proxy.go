package log

import (
	"cmp"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/justwasm/wstack"
)

var (
	corsProxy = cmp.Or(os.Getenv("CRUSH_CORS_PROXY"), "")
	corsUser  = cmp.Or(os.Getenv("CRUSH_CORS_USER"), "crush")
)

// BaseTransport is the original http.DefaultTransport saved before any
// wrapping. Tools that clone the transport should use this.
var BaseTransport = http.DefaultTransport.(*http.Transport)

func init() {
	if corsProxy == "" {
		return
	}

	transport, err := wstack.NewYamuxOverWSTransport(corsProxy)
	if err != nil {
		slog.Warn("Failed to create SSH transport", "error", err)
		return
	}

	http.DefaultTransport = &FallbackTransport{
		Primary:  BaseTransport,
		Fallback: transport,
	}
}

// NewHTTPClientWithTimeout returns an http.Client with the given timeout and
// the patched http.DefaultTransport (which includes CORS proxy retry when
// CRUSH_CORS_PROXY is set).
func NewHTTPClientWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
