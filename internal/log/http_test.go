package log

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestHTTPRoundTripLogger(t *testing.T) {
	// Create a test server that returns a 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error", "code": 500}`))
	}))
	defer server.Close()

	// Create HTTP client with logging
	client := NewHTTPClient(HTTPClientOptions{Debug: true})

	// Make a request
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		server.URL,
		strings.NewReader(`{"test": "data"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}
}

func TestRetryProxyRoundTripperRetriesErrorResponsesThroughProxy(t *testing.T) {
	t.Parallel()

	var urls []string
	client := NewHTTPClient(HTTPClientOptions{
		RetryProxyPrefix: "https://no-cors.deno.dev/",
	})
	client.Transport = &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			urls = append(urls, req.URL.String())
			if strings.HasPrefix(req.URL.String(), "https://no-cors.deno.dev/") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       http.NoBody,
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		},
	}
	client.Transport = WrapHTTPTransport(client.Transport, HTTPClientOptions{
		RetryProxyPrefix: "https://no-cors.deno.dev/",
	})

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"https://example.com/v1/chat/completions",
		strings.NewReader(`{"message":"hello"}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}
	if !slices.Equal(urls, []string{
		"https://example.com/v1/chat/completions",
		"https://no-cors.deno.dev/https://example.com/v1/chat/completions",
	}) {
		t.Fatalf("Unexpected request order: %v", urls)
	}
}

func TestRetryProxyRoundTripperRetriesNetworkErrorsThroughProxy(t *testing.T) {
	t.Parallel()

	var urls []string
	transport := WrapHTTPTransport(&roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			urls = append(urls, req.URL.String())
			if strings.HasPrefix(req.URL.String(), "https://no-cors.deno.dev/") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       http.NoBody,
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("dial tcp timeout")
		},
	}, HTTPClientOptions{
		RetryProxyPrefix: "https://no-cors.deno.dev",
	})

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://example.com/v1/models",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}
	if !slices.Equal(urls, []string{
		"https://example.com/v1/models",
		"https://no-cors.deno.dev/https://example.com/v1/models",
	}) {
		t.Fatalf("Unexpected request order: %v", urls)
	}
}

func TestFormatHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer secret-token"},
		"X-API-Key":     []string{"api-key-123"},
		"User-Agent":    []string{"test-agent"},
	}

	formatted := formatHeaders(headers)

	// Check that sensitive headers are redacted
	if formatted["Authorization"][0] != "[REDACTED]" {
		t.Error("Authorization header should be redacted")
	}
	if formatted["X-API-Key"][0] != "[REDACTED]" {
		t.Error("X-API-Key header should be redacted")
	}

	// Check that non-sensitive headers are preserved
	if formatted["Content-Type"][0] != "application/json" {
		t.Error("Content-Type header should be preserved")
	}
	if formatted["User-Agent"][0] != "test-agent" {
		t.Error("User-Agent header should be preserved")
	}
}

type roundTripFunc struct {
	fn func(*http.Request) (*http.Response, error)
}

func (r *roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.fn(req)
}
