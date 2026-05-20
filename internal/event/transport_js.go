//go:build js && wasm

package event

import (
	"net/http"
	"time"
)

var BaseTransport = http.DefaultTransport.(*http.Transport)

type noCORSTransport struct {
	inner http.RoundTripper
}

func (t *noCORSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("js.fetch:mode", "no-cors")
	return t.inner.RoundTrip(req)
}

func newNoCORSClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &noCORSTransport{
			inner: BaseTransport,
		},
	}
}
