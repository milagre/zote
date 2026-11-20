package nginx_ingress

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/profile"
)

func controllerValues(t *testing.T, typ string, args Args) pulumi.Map {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}
	args.Env = e

	p, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	})
	if err != nil {
		t.Fatalf("profile.New: %v", err)
	}

	controller, ok := values(&args, p, DefaultIngressClass)["controller"].(pulumi.Map)
	if !ok {
		t.Fatal("controller: not a pulumi.Map")
	}

	return controller
}

func TestValues_snippetAnnotationsClosedByDefault(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"remote", "local"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			controller := controllerValues(t, typ, Args{})

			if got, ok := controller["allowSnippetAnnotations"]; ok {
				t.Errorf("allowSnippetAnnotations = %v, want unset", got)
			}
			if got, ok := controller["config"]; ok {
				t.Errorf("config = %v, want unset", got)
			}
		})
	}
}

func TestValues_snippetAnnotationsOpenBothGates(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"remote", "local"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			controller := controllerValues(t, typ, Args{AllowSnippetAnnotations: true})

			if got := controller["allowSnippetAnnotations"]; got != pulumi.Bool(true) {
				t.Errorf("allowSnippetAnnotations = %v, want true", got)
			}

			cfg, ok := controller["config"].(pulumi.Map)
			if !ok {
				t.Fatal("config: not a pulumi.Map")
			}
			if got := cfg["annotations-risk-level"]; got != pulumi.String("Critical") {
				t.Errorf("annotations-risk-level = %v, want Critical", got)
			}
		})
	}
}

func TestValues_disableHTTP2(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"remote", "local"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			controller := controllerValues(t, typ, Args{DisableHTTP2: true})

			cfg, ok := controller["config"].(pulumi.Map)
			if !ok {
				t.Fatal("config: not a pulumi.Map")
			}
			if got := cfg["use-http2"]; got != pulumi.String("false") {
				t.Errorf("use-http2 = %v, want \"false\"", got)
			}
		})
	}
}

func TestValues_disableHTTP2KeepsSnippetGates(t *testing.T) {
	t.Parallel()

	controller := controllerValues(t, "remote", Args{AllowSnippetAnnotations: true, DisableHTTP2: true})

	cfg, ok := controller["config"].(pulumi.Map)
	if !ok {
		t.Fatal("config: not a pulumi.Map")
	}
	if got := cfg["use-http2"]; got != pulumi.String("false") {
		t.Errorf("use-http2 = %v, want \"false\"", got)
	}
	if got := cfg["annotations-risk-level"]; got != pulumi.String("Critical") {
		t.Errorf("annotations-risk-level = %v, want Critical", got)
	}
}

func TestArgs_AllowSnippetAnnotations_defaultsClosed(t *testing.T) {
	t.Parallel()

	var a Args
	if a.AllowSnippetAnnotations {
		t.Error("AllowSnippetAnnotations = true, want false")
	}
}
