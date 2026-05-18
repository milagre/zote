// Package endpoint builds [url.URL] values for synchronous in-cluster HTTP references.
package endpoint

import (
	"net"
	"net/url"
)

// HTTP returns an http URL for host with explicit TCP port and path.
// path is normalized with a leading slash; use "/" for the service root.
func HTTP(host, port, path string) url.URL {
	if path == "" {
		path = "/"
	} else if path[0] != '/' {
		path = "/" + path
	}

	return url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   path,
	}
}
