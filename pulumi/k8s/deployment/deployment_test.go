package deployment

import (
	"reflect"
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/profile"
)

// TestSelectKind covers the full truth table of the mode selector: each
// combination of mode inputs must map to exactly one workload kind. This
// is the primary branching rule across the whole package so it is worth
// exhaustive coverage even as the surface is small today.
func TestSelectKind(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want Kind
	}{
		{"no mode fields set picks proc", Mode{}, KindProc},
		{"HTTP set picks http", Mode{HTTP: &HTTPMode{Port: 8080, Health: "/h"}}, KindHTTP},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := selectKind(tc.mode)
			if got != tc.want {
				t.Fatalf("selectKind(%+v) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}

// TestPublicHostnames exercises the hostname composition used by both
// the component's public output and the HTTP workload's ingress rules.
// Order matters (domain-synthesized hosts first, then verbatim veneers)
// because callers rely on it to drive predictable cert-manager TLS
// secret contents.
func TestPublicHostnames(t *testing.T) {
	tests := []struct {
		name      string
		workload  string
		namespace string
		domains   []string
		veneers   []string
		want      []string
	}{
		{"empty inputs produce empty output", "w", "ns", nil, nil, []string{}},
		{"only domains", "svc", "ns", []string{"example.com"}, nil, []string{"svc.ns.example.com"}},
		{
			"multiple domains preserve input order",
			"svc", "ns",
			[]string{"a.example.com", "b.example.com"},
			nil,
			[]string{"svc.ns.a.example.com", "svc.ns.b.example.com"},
		},
		{
			"veneers follow synthesized hostnames",
			"svc", "ns",
			[]string{"example.com"},
			[]string{"alias.example.org"},
			[]string{"svc.ns.example.com", "alias.example.org"},
		},
		{"only veneers", "svc", "ns", nil, []string{"alias.example.org"}, []string{"alias.example.org"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := publicHostnames(tc.workload, tc.namespace, tc.domains, tc.veneers)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("publicHostnames(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestZAMQPUtilizationStat pins the query the zamqp-consumer convenience emits:
// the ambient stats prefix (lowercased env prefix, namespace, workload name) is
// qualified with the zamqp consumer utilization name and sanitized for
// Prometheus. The hyphen in "my-worker" must collapse to "_" (matching the
// series the adapter actually stores), and the result is matched via __name__.
// The composition itself is owned by the zote runtime.
func TestZAMQPUtilizationStat(t *testing.T) {
	e, err := env.New("zote", "prod", "prod", "mars", "/home/mars", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	got := ZAMQPUtilizationStat(e, "apps", "my-worker")
	want := `avg({__name__="app_apps_my_worker_zamqp_consumer_utilization"})`
	if got != want {
		t.Fatalf("ZAMQPUtilizationStat = %q, want %q", got, want)
	}
}

// TestPrivateHostname encodes the Kubernetes in-cluster DNS naming
// convention our workloads rely on. Drift here silently breaks every
// caller that references a workload by its private hostname.
func TestPrivateHostname(t *testing.T) {
	got := privateHostname("svc", "ns")
	want := "svc.ns.svc.cluster.local"
	if got != want {
		t.Fatalf("privateHostname = %q, want %q", got, want)
	}
}

// TestArgsValidate covers the caller-facing validation rules. Each
// case isolates a single missing or invalid input so the error
// message (matched as a substring) is trustworthy at the call site.
func TestArgsValidate(t *testing.T) {
	good := func() *Args {
		e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
		if err != nil {
			t.Fatalf("env.New: %v", err)
		}
		num := profile.IntRange{Min: 1, Max: 1}
		return &Args{
			Env:       e,
			Namespace: "ns",
			Name:      "svc",
			Image:     "example/image",
			Tag:       "v1",
			Profile:   profile.Profile{Num: &num},
		}
	}

	tests := []struct {
		name      string
		mutate    func(*Args)
		wantError string
	}{
		{"happy path proc", func(*Args) {}, ""},
		{"happy path http", func(a *Args) {
			a.Mode.HTTP = &HTTPMode{Port: 8080, Health: "/h"}
		}, ""},
		{"missing name", func(a *Args) { a.Name = "" }, "Name is required"},
		{"missing namespace", func(a *Args) { a.Namespace = "" }, "Namespace is required"},
		{"missing image", func(a *Args) { a.Image = "" }, "Image is required"},
		{"missing tag", func(a *Args) { a.Tag = "" }, "Tag is required"},
		{
			"http mode without port",
			func(a *Args) { a.Mode.HTTP = &HTTPMode{Health: "/h"} },
			"Mode.HTTP.Port is required",
		},
		{
			"http mode without health path",
			func(a *Args) { a.Mode.HTTP = &HTTPMode{Port: 8080} },
			"Mode.HTTP.Health is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := good()
			tc.mutate(a)
			err := a.validate()
			switch {
			case tc.wantError == "" && err != nil:
				t.Fatalf("validate() = %v, want nil", err)
			case tc.wantError != "" && err == nil:
				t.Fatalf("validate() = nil, want error %q", tc.wantError)
			case tc.wantError != "" && !strings.Contains(err.Error(), tc.wantError):
				t.Fatalf("validate() = %v, want error containing %q", err, tc.wantError)
			}
		})
	}
}

// TestArgsValidateWrapsEnvError confirms that invalid env fields are
// surfaced through Args.validate. Args.validate delegates env checking
// to env.Validate, so this test is the contract that we keep the
// delegation hooked up.
func TestArgsValidateWrapsEnvError(t *testing.T) {
	a := &Args{
		Namespace: "ns",
		Name:      "svc",
		Image:     "i",
		Tag:       "t",
	}
	if err := a.validate(); err == nil {
		t.Fatal("validate() = nil, want error from env.Validate")
	}
}
