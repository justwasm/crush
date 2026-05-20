package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClientOptions struct {
	Debug            bool
	RetryProxyPrefix string
}

// NewHTTPClient creates an HTTP client with debug logging enabled when debug mode is on.
func NewHTTPClient(opts HTTPClientOptions) *http.Client {
	return &http.Client{
		Transport: WrapHTTPTransport(http.DefaultTransport, opts),
	}
}

// WrapHTTPTransport applies shared HTTP behaviors for API clients.
func WrapHTTPTransport(transport http.RoundTripper, opts HTTPClientOptions) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if opts.Debug {
		transport = &HTTPRoundTripLogger{Transport: transport}
	}
	if opts.RetryProxyPrefix != "" {
		transport = &RetryProxyRoundTripper{
			Transport:   transport,
			ProxyPrefix: normalizeProxyPrefix(opts.RetryProxyPrefix),
		}
	}
	return transport
}

// HTTPRoundTripLogger is an http.RoundTripper that logs requests and responses.
type HTTPRoundTripLogger struct {
	Transport http.RoundTripper
}

// RetryProxyRoundTripper retries failed requests once through a trusted proxy prefix.
type RetryProxyRoundTripper struct {
	Transport   http.RoundTripper
	ProxyPrefix string
}

// RoundTrip implements http.RoundTripper interface with logging.
func (h *HTTPRoundTripLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	var err error
	var save io.ReadCloser
	save, req.Body, err = drainBody(req.Body)
	if err != nil {
		slog.Error(
			"HTTP request failed",
			"method", req.Method,
			"url", req.URL,
			"error", err,
		)
		return nil, err
	}

	if slog.Default().Enabled(req.Context(), slog.LevelDebug) {
		slog.Debug(
			"HTTP Request",
			"method", req.Method,
			"url", req.URL,
			"body", bodyToString(save),
		)
	}

	start := time.Now()
	resp, err := h.Transport.RoundTrip(req)
	duration := time.Since(start)
	if err != nil {
		slog.Error(
			"HTTP request failed",
			"method", req.Method,
			"url", req.URL,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return resp, err
	}

	save, resp.Body, err = drainBody(resp.Body)
	if err != nil {
		slog.Error("Failed to drain response body", "error", err)
		return resp, err
	}
	if slog.Default().Enabled(req.Context(), slog.LevelDebug) {
		slog.Debug(
			"HTTP Response",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"headers", formatHeaders(resp.Header),
			"body", bodyToString(save),
			"content_length", resp.ContentLength,
			"duration_ms", duration.Milliseconds(),
		)
	}
	return resp, nil
}

// RoundTrip implements http.RoundTripper interface with a single retry through
// a configured proxy prefix.
func (h *RetryProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("HTTP request is nil")
	}
	if req.URL == nil {
		return nil, fmt.Errorf("HTTP request URL is nil")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return h.Transport.RoundTrip(req)
	}
	if strings.HasPrefix(req.URL.String(), h.ProxyPrefix) {
		return h.Transport.RoundTrip(req)
	}

	body, err := snapshotRequestBody(req)
	if err != nil {
		return nil, err
	}

	resp, err := h.Transport.RoundTrip(cloneRequestWithBody(req, body))
	if !shouldRetryViaProxy(resp, err) {
		return resp, err
	}

	proxyURL, proxyErr := buildProxyURL(h.ProxyPrefix, req.URL)
	if proxyErr != nil {
		slog.Error("Failed to build proxy retry URL", "url", req.URL, "proxy_prefix", h.ProxyPrefix, "error", proxyErr)
		return resp, err
	}

	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	retryReq := cloneRequestWithBody(req, body)
	retryReq.URL = proxyURL
	retryReq.Host = ""

	attrs := []any{
		"method", req.Method,
		"url", req.URL.String(),
		"proxy_url", proxyURL.String(),
	}
	if resp != nil {
		attrs = append(attrs, "status_code", resp.StatusCode)
	}
	if err != nil {
		attrs = append(attrs, "error", err)
	}
	slog.Warn("Retrying HTTP request through proxy", attrs...)

	return h.Transport.RoundTrip(retryReq)
}

func bodyToString(body io.ReadCloser) string {
	if body == nil {
		return ""
	}
	src, err := io.ReadAll(body)
	if err != nil {
		slog.Error("Failed to read body", "error", err)
		return ""
	}
	var b bytes.Buffer
	if json.Indent(&b, bytes.TrimSpace(src), "", "  ") != nil {
		// not json probably
		return string(src)
	}
	return b.String()
}

// formatHeaders formats HTTP headers for logging, filtering out sensitive information.
func formatHeaders(headers http.Header) map[string][]string {
	filtered := make(map[string][]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		// Filter out sensitive headers
		if strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "api-key") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "secret") {
			filtered[key] = []string{"[REDACTED]"}
		} else {
			filtered[key] = values
		}
	}
	return filtered
}

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func snapshotRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	if err := req.Body.Close(); err != nil {
		return nil, fmt.Errorf("failed to close request body: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))

	return body, nil
}

func cloneRequestWithBody(req *http.Request, body []byte) *http.Request {
	clone := req.Clone(req.Context())
	if body == nil {
		clone.Body = http.NoBody
		clone.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		clone.ContentLength = 0
		return clone
	}

	clone.Body = io.NopCloser(bytes.NewReader(body))
	clone.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	clone.ContentLength = int64(len(body))
	return clone
}

func shouldRetryViaProxy(resp *http.Response, err error) bool {
	return err != nil || (resp != nil && resp.StatusCode >= http.StatusBadRequest)
}

func normalizeProxyPrefix(prefix string) string {
	if strings.HasSuffix(prefix, "/") {
		return prefix
	}
	return prefix + "/"
}

func buildProxyURL(prefix string, target *url.URL) (*url.URL, error) {
	return url.Parse(prefix + target.String())
}
