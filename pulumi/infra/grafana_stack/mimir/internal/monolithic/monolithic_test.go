package monolithic

import (
	"strings"
	"testing"
)

func TestConfigYAML_persistsAndExposesRecentBlocks(t *testing.T) {
	t.Parallel()

	got := configYAML("minio:9000", true, "prod-mimir")

	// Each key guards against a distinct way recent samples go missing: an
	// unflushed head on shutdown, a querier that only asks ingesters, and a
	// store-gateway that skips young blocks.
	want := []string{
		"flush_blocks_on_shutdown: true",
		"query_store_after: 0s",
		"ignore_blocks_within: 0s",
		"dir: /data/ingester",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("configYAML missing %q in:\n%s", w, got)
		}
	}
}

func TestConfigYAML_rendersObjectStorage(t *testing.T) {
	t.Parallel()

	got := configYAML("minio:9000", false, "prod-mimir")

	want := []string{
		"endpoint: minio:9000",
		"insecure: false",
		"bucket_name: prod-mimir",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("configYAML missing %q in:\n%s", w, got)
		}
	}
}

func TestStorageSize_defaultsWhenUnset(t *testing.T) {
	t.Parallel()

	if got := storageSize(nil); got != defaultStorageSize {
		t.Errorf("storageSize(nil) = %q, want %q", got, defaultStorageSize)
	}

	empty := ""
	if got := storageSize(&empty); got != defaultStorageSize {
		t.Errorf("storageSize(\"\") = %q, want %q", got, defaultStorageSize)
	}
}

func TestStorageSize_honoursOverride(t *testing.T) {
	t.Parallel()

	size := "50Gi"
	if got := storageSize(&size); got != size {
		t.Errorf("storageSize(%q) = %q, want %q", size, got, size)
	}
}
