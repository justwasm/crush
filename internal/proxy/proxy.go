package proxy

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/log"
)

// proxyDomains caches hostnames that failed direct requests. Subsequent
// requests to these hosts go directly through the proxy.
var proxyDomains sync.Map

// CORSProxyEnv is the environment variable for the CORS proxy prefix used to
// retry failed requests.
const CORSProxyEnv = "CRUSH_CORS_PROXY"

// ResetProxyDomains clears the cached proxy domains. Used in testing.
func ResetProxyDomains() {
	proxyDomains = sync.Map{}
}

// RetryWithProxyTransport wraps an http.RoundTripper to retry failed requests
// through a proxy URL. If the initial request fails (network error or 5xx
// response), the original URL is prefixed with the proxy URL and retried once.
// Domains that fail are cached so subsequent requests skip the direct attempt.
type RetryWithProxyTransport struct {
	Transport http.RoundTripper
	ProxyURL  string
}

// RoundTrip implements http.RoundTripper.
func (t *RetryWithProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host

	// If this domain previously failed, go straight to proxy.
	if _, ok := proxyDomains.Load(host); ok {
		return t.proxyRoundTrip(req)
	}

	// Buffer the body so we can retry through proxy if direct fails.
	var (
		body []byte
		err  error
	)
	if req.Body != nil && req.Body != http.NoBody {
		body, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Try direct first.
	resp, err := t.Transport.RoundTrip(req)
	if err == nil && resp.StatusCode < 500 {
		return resp, nil
	}

	// Record the failure and retry through proxy, restoring the body
	// since the direct attempt consumed it.
	proxyDomains.Store(host, struct{}{})
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return t.proxyRoundTrip(req)
}

// proxyRoundTrip clones the request and sends it through the proxy URL.
func (t *RetryWithProxyTransport) proxyRoundTrip(req *http.Request) (*http.Response, error) {
	proxyURL, parseErr := url.Parse(t.ProxyURL)
	if parseErr != nil {
		return nil, parseErr
	}

	proxyReq := req.Clone(req.Context())
	proxyReq.URL = proxyURL.JoinPath(req.URL.String())

	slog.Warn("Retrying request through proxy",
		"original_url", req.URL,
		"proxy_url", proxyReq.URL,
	)

	return t.Transport.RoundTrip(proxyReq)
}

// newHTTPClientWithProxy creates an HTTP client that retries failed requests
// through the given proxy URL. If debug is true, requests are also logged.
func newHTTPClientWithProxy(proxyURL string, debug bool) *http.Client {
	var transport http.RoundTripper = http.DefaultTransport
	if debug {
		transport = &log.HTTPRoundTripLogger{Transport: transport}
	}
	transport = &RetryWithProxyTransport{
		Transport: transport,
		ProxyURL:  proxyURL,
	}
	return &http.Client{Transport: transport}
}

// ProviderHTTPClient returns an *http.Client configured from the
// CRUSH_CORS_PROXY environment variable and/or debug logging.
// Returns nil when neither is needed.
func ProviderHTTPClient(debug bool) *http.Client {
	proxyURL := os.Getenv(CORSProxyEnv)
	if proxyURL == "" && !debug {
		return nil
	}
	if proxyURL != "" {
		return newHTTPClientWithProxy(proxyURL, debug)
	}
	return log.NewHTTPClient()
}

// NewHTTPClientFromEnv creates an HTTP client configured from the
// CRUSH_CORS_PROXY environment variable with the given timeout.
// Returns a plain client if the env var is not set.
func NewHTTPClientFromEnv(timeout time.Duration) *http.Client {
	proxyURL := os.Getenv(CORSProxyEnv)
	if proxyURL == "" {
		return &http.Client{Timeout: timeout}
	}
	client := newHTTPClientWithProxy(proxyURL, false)
	client.Timeout = timeout
	return client
}
