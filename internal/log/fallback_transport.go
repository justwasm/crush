package log

import (
	"io"
	"net/http"
	"sync"
)

// proxyDomains caches hostnames that failed direct requests. Subsequent
// requests to these hosts go directly through the proxy.
var proxyDomains sync.Map

// ResetProxyDomains clears the cached proxy domains. Used in testing.
func ResetProxyDomains() {
	proxyDomains = sync.Map{}
}

// FallbackTransport tries the primary RoundTripper first. If it returns an
// error or a 5xx response, the request is retried once with the fallback.
//
// When CRUSH_CORS_PROXY is set, this is installed as http.DefaultTransport so
// that requests that fail due to CORS are transparently retried through the
// configured proxy prefix.
type FallbackTransport struct {
	Primary  http.RoundTripper
	Fallback http.RoundTripper
}

func (t *FallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host

	// If this domain previously failed, go straight to proxy.
	if _, ok := proxyDomains.Load(host); ok {
		return t.Fallback.RoundTrip(req)
	}

	// Buffer the body so we can retry through proxy if direct fails.
	var retryBody io.ReadCloser
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		req.Body, retryBody, err = drainBody(req.Body)
		if err != nil {
			return nil, err
		}
	}

	// Try primary first.
	resp, err := t.Primary.RoundTrip(req)
	if err == nil && resp.StatusCode < 500 {
		return resp, nil
	}

	// Cache the failure and retry through proxy. Restore the body since the
	// primary transport may have consumed it.
	proxyDomains.Store(host, struct{}{})
	if retryBody != nil {
		req.Body = retryBody
	}

	return t.Fallback.RoundTrip(req)
}
