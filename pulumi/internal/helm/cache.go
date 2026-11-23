package helm

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/milagre/zote/go/zhttpclient"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

// CacheDir is where charts are kept, relative to the Pulumi project root.
// Consumers should ignore it in version control.
const CacheDir = ".charts"

var defaultCache = NewChartCache()

// LocalChart returns the Release chart input for chart at version from
// repository, downloading it into [CacheDir] under the project root first.
// The Release must not also set RepositoryOpts. An empty repository means
// chart is a full OCI reference, returned unchanged.
func LocalChart(ctx *pulumi.Context, repository, chart, version string) (pulumi.StringInput, error) {
	if repository == "" {
		return pulumi.String(chart), nil
	}

	abs, err := defaultCache.Fetch(ctx.Context(), ctx.RootDirectory(), repository, chart, version)
	if err != nil {
		return nil, fmt.Errorf("caching chart %q: %w", chart, err)
	}

	rel, err := workdirRelative(abs)
	if err != nil {
		return nil, fmt.Errorf("caching chart %q: %w", chart, err)
	}

	return pulumi.String(rel), nil
}

// LocalFile returns the path of the file at fileURL, downloading it into
// [CacheDir] under the project root first. digest is the file's sha256 in
// hex, which a download must match. fileURL must name content that never
// changes, since a cached file is never refetched.
func LocalFile(ctx *pulumi.Context, fileURL, digest string) (string, error) {
	abs, err := defaultCache.FetchFile(ctx.Context(), ctx.RootDirectory(), fileURL, digest)
	if err != nil {
		return "", fmt.Errorf("caching %s: %w", fileURL, err)
	}

	rel, err := workdirRelative(abs)
	if err != nil {
		return "", fmt.Errorf("caching %s: %w", fileURL, err)
	}

	return rel, nil
}

// ChartCache keeps chart archives on disk so a helm/v3 Release renders from a
// local file. The Release provider otherwise downloads its chart on every
// preview, without retry, so a stack with many charts fails whenever any one
// repository is slow.
//
// A cached archive is never refetched: chart versions are immutable.
type ChartCache struct {
	Client   *http.Client
	Attempts int
	// Backoff is the wait after the first failed attempt, doubled after each one.
	Backoff time.Duration

	mu      sync.Mutex
	indexes map[string]*repoIndex
}

func NewChartCache() *ChartCache {
	opts := zhttpclient.DefaultOptions()
	opts.Timeouts.TimeLimit = 2 * time.Minute
	opts.Timeouts.HTTPTimeouts.ResponseHeaderTimeout = 15 * time.Second
	opts.TLSConfig.MinVersion = tls.VersionTLS12

	return &ChartCache{
		Client:   zhttpclient.New(opts),
		Attempts: 5,
		Backoff:  time.Second,
		indexes:  map[string]*repoIndex{},
	}
}

// Fetch returns the archive's path under root/[CacheDir], downloading it if absent.
func (c *ChartCache) Fetch(ctx context.Context, root, repository, chart, version string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version is required")
	}

	repoURL, err := url.Parse(strings.TrimSuffix(repository, "/") + "/")
	if err != nil {
		return "", fmt.Errorf("parsing repository %q: %w", repository, err)
	}

	dir := filepath.Join(root, CacheDir, repoURL.Host, filepath.FromSlash(path.Clean(repoURL.Path)))
	archiveName := func(v string) string {
		return chart + "-" + strings.TrimPrefix(v, "v") + ".tgz"
	}

	// An exact version can be served from disk without the index; a range
	// has to consult it, since a newer match may have been published.
	if _, err := semver.StrictNewVersion(strings.TrimPrefix(version, "v")); err == nil {
		dest := filepath.Join(dir, archiveName(version))
		hit, err := exists(dest)
		if err != nil {
			return "", err
		}
		if hit {
			return dest, nil
		}
	}

	entry, err := c.lookup(ctx, repoURL, chart, version)
	if err != nil {
		return "", err
	}

	dest := filepath.Join(dir, archiveName(entry.Version))
	hit, err := exists(dest)
	if err != nil {
		return "", err
	}
	if hit {
		return dest, nil
	}
	if len(entry.URLs) == 0 {
		return "", fmt.Errorf("%s %s has no download url in %s", chart, version, repoURL)
	}

	archiveURL, err := repoURL.Parse(entry.URLs[0])
	if err != nil {
		return "", fmt.Errorf("parsing chart url %q: %w", entry.URLs[0], err)
	}

	body, err := c.get(ctx, archiveURL.String())
	if err != nil {
		return "", fmt.Errorf("downloading %s %s: %w", chart, version, err)
	}

	if entry.Digest != "" {
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, entry.Digest) {
			return "", fmt.Errorf("%s %s: digest %s does not match index digest %s", chart, version, got, entry.Digest)
		}
	}

	err = writeAtomic(dest, body)
	if err != nil {
		return "", fmt.Errorf("caching %s %s: %w", chart, version, err)
	}

	return dest, nil
}

// FetchFile returns the path of the file at fileURL under root/[CacheDir],
// downloading it if absent. digest is the file's sha256 in hex.
func (c *ChartCache) FetchFile(ctx context.Context, root, fileURL, digest string) (string, error) {
	if digest == "" {
		return "", fmt.Errorf("digest is required")
	}

	u, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("parsing url %q: %w", fileURL, err)
	}

	dest := filepath.Join(root, CacheDir, u.Host, filepath.FromSlash(path.Clean("/"+u.Path)))
	hit, err := exists(dest)
	if err != nil {
		return "", err
	}
	if hit {
		return dest, nil
	}

	body, err := c.get(ctx, fileURL)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", fileURL, err)
	}

	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, digest) {
		return "", fmt.Errorf("%s: digest %s does not match %s", fileURL, got, digest)
	}

	err = writeAtomic(dest, body)
	if err != nil {
		return "", fmt.Errorf("caching %s: %w", fileURL, err)
	}

	return dest, nil
}

// lookup picks the highest version satisfying version as a semver
// constraint, which is how Helm resolves a chart version against an index.
func (c *ChartCache) lookup(ctx context.Context, repoURL *url.URL, chart, version string) (indexEntry, error) {
	constraint, err := semver.NewConstraint(version)
	if err != nil {
		return indexEntry{}, fmt.Errorf("parsing version %q: %w", version, err)
	}

	index, err := c.index(ctx, repoURL)
	if err != nil {
		return indexEntry{}, err
	}

	var best indexEntry
	var bestVersion *semver.Version
	for _, e := range index.Entries[chart] {
		v, err := semver.NewVersion(e.Version)
		if err != nil || !constraint.Check(v) {
			continue
		}
		if bestVersion == nil || v.GreaterThan(bestVersion) {
			best, bestVersion = e, v
		}
	}
	if bestVersion == nil {
		return indexEntry{}, fmt.Errorf("%s %s not found in %s", chart, version, repoURL)
	}

	return best, nil
}

// index is memoized per repository: several charts can share one, and some
// indexes run to tens of megabytes.
func (c *ChartCache) index(ctx context.Context, repoURL *url.URL) (*repoIndex, error) {
	key := repoURL.String()

	c.mu.Lock()
	defer c.mu.Unlock()

	if idx, ok := c.indexes[key]; ok {
		return idx, nil
	}

	body, err := c.get(ctx, key+"index.yaml")
	if err != nil {
		return nil, fmt.Errorf("fetching index for %s: %w", key, err)
	}

	idx := &repoIndex{}
	err = yaml.Unmarshal(body, idx)
	if err != nil {
		return nil, fmt.Errorf("parsing index for %s: %w", key, err)
	}

	if c.indexes == nil {
		c.indexes = map[string]*repoIndex{}
	}
	c.indexes[key] = idx

	return idx, nil
}

func (c *ChartCache) get(ctx context.Context, u string) ([]byte, error) {
	attempts := max(c.Attempts, 1)
	wait := c.Backoff

	var lastErr error
	for i := range attempts {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("waiting to retry: %w", ctx.Err())
			case <-time.After(wait):
			}
			wait *= 2
		}

		body, err := c.getOnce(ctx, u)
		if err == nil {
			return body, nil
		}

		var se *statusError
		if errors.As(err, &se) && !se.retryable() {
			return nil, err
		}

		lastErr = err
	}

	return nil, fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

func (c *ChartCache) getOnce(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("requesting %s: %w", u, &statusError{code: resp.StatusCode})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", u, err)
	}

	return body, nil
}

type repoIndex struct {
	Entries map[string][]indexEntry `yaml:"entries"`
}

type indexEntry struct {
	Version string   `yaml:"version"`
	URLs    []string `yaml:"urls"`
	Digest  string   `yaml:"digest"`
}

type statusError struct {
	code int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("status %d", e.code)
}

func (e *statusError) retryable() bool {
	return e.code == http.StatusTooManyRequests || e.code >= 500
}

func exists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("checking cache: %w", err)
}

// writeAtomic renames into place so a concurrent or interrupted run never
// sees a partial archive, which the cache would otherwise trust forever.
func writeAtomic(dest string, body []byte) error {
	err := os.MkdirAll(filepath.Dir(dest), 0o755)
	if err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".partial-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.Write(body)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	err = tmp.Close()
	if err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	err = os.Rename(tmp.Name(), dest)
	if err != nil {
		return fmt.Errorf("renaming into cache: %w", err)
	}

	return nil
}

// workdirRelative is relative because the path lands in resource state, where
// an absolute one differs between machines and diffs on every deploy. Pulumi
// starts the provider in the program's working directory, so it resolves the
// same way.
func workdirRelative(abs string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	rel, err := filepath.Rel(wd, abs)
	if err != nil {
		return "", fmt.Errorf("relativizing %s: %w", abs, err)
	}

	return dotRelative(rel), nil
}

// dotRelative prefixes "./" so the provider reads p as a path; a bare
// "dir/x.tgz" is otherwise taken as a repo/chart reference when missing.
func dotRelative(p string) string {
	if strings.HasPrefix(p, ".") {
		return p
	}

	return "." + string(filepath.Separator) + p
}
