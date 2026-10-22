package helm_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/milagre/zote/pulumi/internal/helm"
)

const archive = "chart archive bytes"

type repo struct {
	srv       *httptest.Server
	requests  atomic.Int32
	failFirst atomic.Int32
}

// newRepo serves an index listing my-chart 1.2.3, whose url is chartURL
// resolved against the server (relative, or absolute when it starts with "/").
func newRepo(t *testing.T, chartURL, digest string) *repo {
	t.Helper()

	r := &repo{}
	r.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.requests.Add(1)
		if r.failFirst.Load() > 0 {
			r.failFirst.Add(-1)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		switch req.URL.Path {
		case "/charts/index.yaml":
			u := chartURL
			if strings.HasPrefix(u, "/") {
				u = "https://" + req.Host + u
			}
			fmt.Fprintf(w, "apiVersion: v1\nentries:\n  my-chart:\n  - version: 1.2.3\n    digest: %s\n    urls:\n    - %s\n  - version: 1.0.0\n    urls:\n    - old.tgz\n", digest, u)
		case "/charts/my-chart-1.2.3.tgz", "/releases/my-chart-1.2.3.tgz":
			fmt.Fprint(w, archive)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(r.srv.Close)

	return r
}

func newCache(r *repo) *helm.ChartCache {
	c := helm.NewChartCache()
	c.Client = r.srv.Client()
	c.Backoff = 0

	return c
}

func archiveDigest() string {
	sum := sha256.Sum256([]byte(archive))
	return hex.EncodeToString(sum[:])
}

func TestChartCache_Fetch_downloadsUnderRoot(t *testing.T) {
	t.Parallel()

	for name, chartURL := range map[string]string{
		"relative url": "my-chart-1.2.3.tgz",
		"absolute url": "/releases/my-chart-1.2.3.tgz",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := newRepo(t, chartURL, archiveDigest())
			root := t.TempDir()

			path, err := newCache(r).Fetch(context.Background(), root, r.srv.URL+"/charts", "my-chart", "1.2.3")
			if err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if !strings.HasPrefix(path, filepath.Join(root, helm.CacheDir)+string(filepath.Separator)) {
				t.Errorf("path %q not under cache dir", path)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading cached chart: %v", err)
			}
			if string(got) != archive {
				t.Errorf("cached chart = %q", got)
			}
		})
	}
}

func TestChartCache_Fetch_hitMakesNoRequests(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", "")
	root := t.TempDir()
	repoURL := r.srv.URL + "/charts"

	first, err := newCache(r).Fetch(context.Background(), root, repoURL, "my-chart", "1.2.3")
	if err != nil {
		t.Fatalf("first Fetch: %v", err)
	}
	before := r.requests.Load()

	second, err := newCache(r).Fetch(context.Background(), root, repoURL, "my-chart", "1.2.3")
	if err != nil {
		t.Fatalf("second Fetch: %v", err)
	}
	if second != first {
		t.Errorf("second path %q != first %q", second, first)
	}
	if n := r.requests.Load() - before; n != 0 {
		t.Errorf("cache hit made %d requests", n)
	}
}

func TestChartCache_Fetch_retriesFailedRequests(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", archiveDigest())
	r.failFirst.Store(2)

	if _, err := newCache(r).Fetch(context.Background(), t.TempDir(), r.srv.URL+"/charts", "my-chart", "1.2.3"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
}

func TestChartCache_Fetch_givesUpAfterAttempts(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", archiveDigest())
	r.failFirst.Store(100)
	c := newCache(r)
	c.Attempts = 3

	if _, err := c.Fetch(context.Background(), t.TempDir(), r.srv.URL+"/charts", "my-chart", "1.2.3"); err == nil {
		t.Fatal("expected error")
	}
	if n := r.requests.Load(); n != 3 {
		t.Errorf("requests = %d, want 3", n)
	}
}

func TestChartCache_Fetch_rejectsDigestMismatch(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", strings.Repeat("0", 64))
	root := t.TempDir()

	if _, err := newCache(r).Fetch(context.Background(), root, r.srv.URL+"/charts", "my-chart", "1.2.3"); err == nil {
		t.Fatal("expected error")
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			t.Errorf("left file behind: %s", path)
		}

		return err
	})
	if err != nil {
		t.Fatalf("walking root: %v", err)
	}
}

func TestChartCache_Fetch_unknownVersion(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", "")

	if _, err := newCache(r).Fetch(context.Background(), t.TempDir(), r.srv.URL+"/charts", "my-chart", "9.9.9"); err == nil {
		t.Fatal("expected error")
	}
}

func TestChartCache_Fetch_resolvesVersionsLikeHelm(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"v1.2.3", "1", "^1.0.0"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			r := newRepo(t, "my-chart-1.2.3.tgz", archiveDigest())

			path, err := newCache(r).Fetch(context.Background(), t.TempDir(), r.srv.URL+"/charts", "my-chart", version)
			if err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if filepath.Base(path) != "my-chart-1.2.3.tgz" {
				t.Errorf("path %q: want the 1.2.3 archive", path)
			}
		})
	}
}

func TestChartCache_Fetch_rangeRechecksIndex(t *testing.T) {
	t.Parallel()

	r := newRepo(t, "my-chart-1.2.3.tgz", "")
	root := t.TempDir()
	repoURL := r.srv.URL + "/charts"

	_, err := newCache(r).Fetch(context.Background(), root, repoURL, "my-chart", "1")
	if err != nil {
		t.Fatalf("first Fetch: %v", err)
	}
	before := r.requests.Load()

	_, err = newCache(r).Fetch(context.Background(), root, repoURL, "my-chart", "1")
	if err != nil {
		t.Fatalf("second Fetch: %v", err)
	}
	if n := r.requests.Load() - before; n != 1 {
		t.Errorf("requests = %d, want 1 (index only)", n)
	}
}
