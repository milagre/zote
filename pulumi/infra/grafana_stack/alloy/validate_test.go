package alloy

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestValidateArgs_rejectsNilRiver(t *testing.T) {
	t.Parallel()

	err := validateArgs(&Args{
		Namespace: "ns",
		Config:    Config{},
		River:     nil,
	})
	if err == nil {
		t.Fatal("expected error for nil River")
	}
}

func TestValidateArgs_acceptsRiver(t *testing.T) {
	t.Parallel()

	err := validateArgs(&Args{
		Namespace: "ns",
		Config:    Config{},
		River:     pulumi.String(`logging { level = "info" }`),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateArgs_rejectsNilArgs(t *testing.T) {
	t.Parallel()

	err := validateArgs(nil)
	if err == nil {
		t.Fatal("expected error for nil args")
	}
}
