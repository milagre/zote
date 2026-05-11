package endpoint

import (
	"net/url"
	"testing"
)

func TestHTTP_joinsHostPortAndPath(t *testing.T) {
	t.Parallel()

	u := HTTP("loki-gateway.ns.svc.cluster.local", "80", "/loki/api/v1/push")
	if u.Scheme != "http" {
		t.Fatalf("Scheme: got %q", u.Scheme)
	}
	if u.Hostname() != "loki-gateway.ns.svc.cluster.local" {
		t.Fatalf("Hostname: got %q", u.Hostname())
	}
	if u.Port() != "80" {
		t.Fatalf("Port: got %q", u.Port())
	}
	if u.Path != "/loki/api/v1/push" {
		t.Fatalf("Path: got %q", u.Path)
	}
}

func TestHTTP_emptyPathBecomesSlash(t *testing.T) {
	t.Parallel()

	u := HTTP("svc.ns.svc.cluster.local", "80", "")
	if u.Path != "/" {
		t.Fatalf("Path: got %q", u.Path)
	}
}

func TestHTTP_addsLeadingSlashToPath(t *testing.T) {
	t.Parallel()

	u := HTTP("h", "80", "prometheus")
	if u.Path != "/prometheus" {
		t.Fatalf("Path: got %q", u.Path)
	}
}

func TestHTTP_roundTripStringParse(t *testing.T) {
	t.Parallel()

	orig := HTTP("mimir-nginx.ns.svc.cluster.local", "80", "/api/v1/push")
	parsed, err := url.Parse(orig.String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != orig.Scheme || parsed.Host != orig.Host || parsed.Path != orig.Path {
		t.Fatalf("parse mismatch: got %#v want %#v", parsed, orig)
	}
}
