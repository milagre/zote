package config_test

import (
	"testing"
	"testing/fstest"

	"github.com/milagre/zote/pulumi/config"
	"github.com/milagre/zote/pulumi/env"
)

func mustEnv(t *testing.T) env.Env {
	t.Helper()

	e, err := env.New("local", "dev", "username", "/tmp", "")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func TestLoader_ordering_and_merge(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"env/local/1.main.yaml": {Data: []byte(`
a: from-type
nested:
  x: 1
  y: 2
`)},
		"env/local/dev/1.main.yaml": {Data: []byte(`
a: from-tier
nested:
  y: 20
  z: 30
`)},
		"env/local/dev/username/1.main.yaml": {Data: []byte(`
a: from-name
nested:
  z: 300
`)},
		"env/local/dev/username/5.info.yaml": {Data: []byte(`info: env-${env.name}`)},
		"env/local/dev/username/_9.secrets.yaml": {Data: []byte(`ignored: yes`)},
	}

	l := &config.Loader{FS: fsys, Env: mustEnv(t)}
	got, err := l.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantSources := []string{
		"env/local/1.main.yaml",
		"env/local/dev/1.main.yaml",
		"env/local/dev/username/1.main.yaml",
		"env/local/dev/username/5.info.yaml",
	}
	if len(got.Sources) != len(wantSources) {
		t.Fatalf("Sources: got %v, want %v", got.Sources, wantSources)
	}
	for i, src := range wantSources {
		if got.Sources[i] != src {
			t.Errorf("Sources[%d]: got %q, want %q", i, got.Sources[i], src)
		}
	}

	if got.Data["a"] != "from-name" {
		t.Errorf("a: got %v, want from-name", got.Data["a"])
	}

	nested, ok := got.Data["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested: got %T", got.Data["nested"])
	}
	if nested["x"] != 1 {
		t.Errorf("nested.x: got %v, want 1", nested["x"])
	}
	if nested["y"] != 20 {
		t.Errorf("nested.y: got %v, want 20", nested["y"])
	}
	if nested["z"] != 300 {
		t.Errorf("nested.z: got %v, want 300", nested["z"])
	}

	if got.Data["info"] != "env-username" {
		t.Errorf("info: got %v, want env-username", got.Data["info"])
	}
}

func TestLoader_missingFolderIsOK(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"env/local/1.main.yaml": {Data: []byte(`a: only-here`)},
	}

	l := &config.Loader{FS: fsys, Env: mustEnv(t)}
	got, err := l.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Data["a"] != "only-here" {
		t.Errorf("a: got %v, want only-here", got.Data["a"])
	}
	if len(got.Sources) != 1 {
		t.Errorf("Sources: got %v, want one entry", got.Sources)
	}
}

func TestLoader_badTemplate(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"env/local/1.main.yaml": {Data: []byte(`bad: ${env.nope}`)},
	}

	l := &config.Loader{FS: fsys, Env: mustEnv(t)}
	if _, err := l.Load(); err == nil {
		t.Fatal("expected error on unknown env field")
	}
}

func TestLoader_escapedPlaceholder(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"env/local/1.main.yaml": {Data: []byte(`literal: "$${env.name}"`)},
	}

	l := &config.Loader{FS: fsys, Env: mustEnv(t)}
	got, err := l.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Data["literal"] != "${env.name}" {
		t.Errorf("literal: got %v, want ${env.name}", got.Data["literal"])
	}
}

func TestLoader_globIgnoresLeadingUnderscore(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"env/local/_9.secrets.yaml": {Data: []byte(`ignored: yes`)},
		"env/local/1.main.yaml":     {Data: []byte(`a: ok`)},
	}

	l := &config.Loader{FS: fsys, Env: mustEnv(t)}
	got, err := l.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, hasIgnored := got.Data["ignored"]; hasIgnored {
		t.Errorf("expected _9.*.yaml to be skipped; data = %v", got.Data)
	}
	if got.Data["a"] != "ok" {
		t.Errorf("a: got %v, want ok", got.Data["a"])
	}
}
