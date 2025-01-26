package zhttpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	_ "github.com/breml/rootcerts"
)

// Options provides centralized discrete options for controlling HTTP request
// settings. DefaultOptions() provides sane defaults for all of these. Do not
// construct this yourself, as that risks not setting some values. Use
// DefaultOptions() and manipulate the result.
type Options struct {
	SecureOnly bool

	Timeouts Timeouts

	TLSConfig *tls.Config
}

type Timeouts struct {
	// TimeLimit is the full round-trip time to connect, make the request, and
	// read the full response. Attempts to read the response body will fail if
	// this time limit is exceeded. This time limit being exceeded is equivalent
	// to providing a context that is canceled after the provided duration. You
	// should use a Context timeout, but this is always set to ensure a limit is
	// provided.
	TimeLimit time.Duration

	HTTPTimeouts HTTPTimeouts

	NetworkTimeouts NetworkTimeouts
}

type HTTPTimeouts struct {
	// ExpectContinueTimeout, if non-zero, specifies the amount of time to wait
	// for a server's first response headers after fully writing the request
	// headers if the request has an "Expect: 100-continue" header. Zero means
	// no timeout and causes the body to be sent immediately, without waiting
	// for the server to approve. This time does not include the time to send
	// the request header.
	ExpectContinueTimeout time.Duration

	// ResponseHeaderTimeout is how long to wait for the response headers from
	// the server after fully writing the request (including its body, if any).
	// Does not include receipt of the full response body or reading the
	// response body. This is your primary timeout to determine how long to wait
	// for the server to reply.
	ResponseHeaderTimeout time.Duration
}

type NetworkTimeouts struct {
	// ConnectionTimeout is the amount of time the client will wait to establish
	// a connection with the destination, excluding TLS
	ConnectionTimeout time.Duration

	// TLSTimeout is the amount of time the client will wait to perform the TLS
	// handshake after the connection is established. Does not overlap with
	// ConnectionTimeout
	TLSTimeout time.Duration
}

func DefaultOptions() Options {
	return Options{
		SecureOnly: true,
		Timeouts: Timeouts{
			TimeLimit: 1 * time.Minute,
			HTTPTimeouts: HTTPTimeouts{
				ExpectContinueTimeout: 5 * time.Second,
			},
			NetworkTimeouts: NetworkTimeouts{
				ConnectionTimeout: 5 * time.Second,
				TLSTimeout:        5 * time.Second,
			},
		},
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}
}

func New(options Options) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()

	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.IdleConnTimeout = 60 * time.Second

	netDialer := &net.Dialer{
		Timeout:   options.Timeouts.NetworkTimeouts.ConnectionTimeout,
		KeepAlive: 2500 * time.Millisecond,
	}

	t.DialContext = netDialer.DialContext
	if options.SecureOnly {
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("insecure http requests are disabled, please use https")
		}
	}

	tlsDialer := &tls.Dialer{
		NetDialer: netDialer,
		Config:    options.TLSConfig.Clone(),
	}
	t.DialTLSContext = tlsDialer.DialContext

	t.TLSHandshakeTimeout = options.Timeouts.NetworkTimeouts.TLSTimeout
	t.ExpectContinueTimeout = options.Timeouts.HTTPTimeouts.ExpectContinueTimeout
	t.ResponseHeaderTimeout = options.Timeouts.HTTPTimeouts.ResponseHeaderTimeout

	client := &http.Client{
		Timeout:   options.Timeouts.TimeLimit,
		Transport: t,
	}

	return client
}
