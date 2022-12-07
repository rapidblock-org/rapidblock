package httpclient

import (
	"net/http"
)

type RequestHeaderRoundTripper struct {
	Next   http.RoundTripper
	Header http.Header
}

func (rt *RequestHeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	a := rt.Header
	aLen := uint(len(a))

	b := req.Header
	bLen := uint(len(b))

	c := make(http.Header, aLen+bLen)
	for key, values := range a {
		c[key] = values
	}
	for key, values := range b {
		c[key] = values
	}

	req = req.WithContext(req.Context())
	req.Header = c
	return rt.Next.RoundTrip(req)
}

var _ http.RoundTripper = (*RequestHeaderRoundTripper)(nil)
