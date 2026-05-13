package helm_test

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
)

func TestValues_acceptsPulumiStringInput(t *testing.T) {
	t.Parallel()

	_ = helm.Values(map[string]any{
		"k": pulumi.String("v"),
	})
}

func TestValues_acceptsPulumiStringOutputLeaf(t *testing.T) {
	t.Parallel()

	out := pulumi.ToOutput("secret").(pulumi.StringOutput)
	_ = helm.Values(map[string]any{
		"token": out,
	})
}

func TestValues_acceptsNestedPulumiInput(t *testing.T) {
	t.Parallel()

	_ = helm.Values(map[string]any{
		"outer": map[string]any{
			"inner": pulumi.String("v"),
		},
	})
}

func TestValues_acceptsPulumiInputInSlice(t *testing.T) {
	t.Parallel()

	_ = helm.Values(map[string]any{
		"list": []any{pulumi.String("a"), "plain"},
	})
}
