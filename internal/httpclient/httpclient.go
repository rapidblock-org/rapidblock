package httpclient

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

func Dialer() *net.Dialer {
	return myDialer
}

func Transport() *http.Transport {
	return myTransport
}

func TransportH2() *http2.Transport {
	return myTransportH2
}

var myDialer = &net.Dialer{
	Timeout:   10 * time.Second,
	KeepAlive: 30 * time.Second,
}

var myTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           myDialer.DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          10,
	MaxIdleConnsPerHost:   2,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var myTransportH2 *http2.Transport

func init() {
	h2, err := http2.ConfigureTransports(myTransport)
	if err != nil {
		panic(err)
	}
	myTransportH2 = h2
}

func Client(rt http.RoundTripper) *http.Client {
	return &http.Client{
		Transport:     rt,
		CheckRedirect: httpNeverRedirect,
		Timeout:       30 * time.Second,
	}
}

func httpNeverRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
