package clientauth

import (
	"fmt"
	"net/http"
)

// BearerTransport injects a bearer token into each request's Authorization
// header before delegating to the wrapped transport.
type BearerTransport struct {
	base          http.RoundTripper
	tokenProvider AccessTokenProvider
}

// NewBearerTransport returns a transport that adds bearer tokens from provider
// to each request. If base is nil, [http.DefaultTransport] is used.
func NewBearerTransport(base http.RoundTripper, provider AccessTokenProvider) *BearerTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &BearerTransport{base: base, tokenProvider: provider}
}

// RoundTrip clones req, sets the Authorization header, and forwards it.
func (t *BearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenProvider.AccessToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	clonedReq := req.Clone(req.Context())
	clonedReq.Header.Set("Authorization", "Bearer "+token)

	return t.base.RoundTrip(clonedReq)
}
