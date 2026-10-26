package cloudflare_tunnel

import (
	"strconv"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// cloudflared releases before 2026.10.2 trust the compromised eMudhra (emSign)
// intermediates.
func TestChartValues_cloudflaredImageTrustsNoCompromisedCAs(t *testing.T) {
	t.Parallel()

	cloudflared, ok := chartValues(&Args{})["cloudflared"].(pulumi.Map)
	if !ok {
		t.Fatalf("cloudflared values missing")
	}
	image, ok := cloudflared["image"].(pulumi.Map)
	if !ok {
		t.Fatalf("cloudflared.image values missing")
	}

	if got := image["repository"]; got != pulumi.String("cloudflare/cloudflared") {
		t.Errorf("repository = %v, want cloudflare/cloudflared", got)
	}

	tag, ok := image["tag"].(pulumi.String)
	if !ok {
		t.Fatalf("tag missing")
	}
	if !atLeast(string(tag), []int{2026, 10, 2}) {
		t.Errorf("tag = %q, want >= 2026.10.2", tag)
	}
}

func atLeast(version string, floor []int) bool {
	parts := strings.Split(version, ".")
	for i, min := range floor {
		if i >= len(parts) {
			return false
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return false
		}
		if n != min {
			return n > min
		}
	}

	return true
}
