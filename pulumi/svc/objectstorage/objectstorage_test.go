package objectstorage_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/svc/objectstorage"
)

func TestObjectStorage_ProvisionedBucket(t *testing.T) {
	t.Parallel()

	o := objectstorage.ObjectStorage{
		Buckets: map[string]string{
			"mimir": "app-prod-infra-objectstorage-mimir",
			"loki":  "app-prod-infra-objectstorage-loki",
		},
	}

	got, err := o.ProvisionedBucket("mimir")
	if err != nil {
		t.Fatalf("ProvisionedBucket: %v", err)
	}
	if want := "app-prod-infra-objectstorage-mimir"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	if _, err := o.ProvisionedBucket("missing"); err == nil {
		t.Fatal("expected error for unknown bucket")
	}
}

func TestObjectStorage_ProvisionedBucket_nilMap(t *testing.T) {
	t.Parallel()

	o := objectstorage.ObjectStorage{}

	if _, err := o.ProvisionedBucket("loki"); err == nil {
		t.Fatal("expected error when Buckets is nil")
	}
}
