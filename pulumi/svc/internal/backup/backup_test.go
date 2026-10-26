package backup_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/svc/internal/backup"
	"github.com/milagre/zote/pulumi/svc/objectstorage"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validSpec() backup.Spec {
	return backup.Spec{
		Storage:  objectstorage.ObjectStorage{Buckets: map[string]string{"dumps": "app-dev-apps-backups-dumps"}},
		Bucket:   "dumps",
		Schedule: "0 7 * * *",
		Profile: profile.Profile{
			CPUCores: profile.FloatRange{Min: 0.1, Max: 0.5},
			MemMB:    profile.IntRange{Min: 128, Max: 512},
		},
	}
}

func TestSpec_Validate_ok(t *testing.T) {
	t.Parallel()

	s := validSpec()
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestSpec_Validate_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*backup.Spec){
		"no schedule":       func(s *backup.Spec) { s.Schedule = "" },
		"no bucket":         func(s *backup.Spec) { s.Bucket = "" },
		"unprovisioned":     func(s *backup.Spec) { s.Bucket = "other" },
		"no memory request": func(s *backup.Spec) { s.Profile.MemMB = profile.IntRange{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := validSpec()
			mutate(&s)
			if err := s.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
