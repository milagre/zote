package digitalocean_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/database/mysql/internal/digitalocean"
)

func TestSpec_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	s := &digitalocean.Spec{
		Primary: digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
	}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestSpec_Validate_rejectsNil(t *testing.T) {
	t.Parallel()

	var s *digitalocean.Spec
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil spec error, got %v", err)
	}
}

func TestSpec_Validate_rejectsMissingPrimaryClass(t *testing.T) {
	t.Parallel()

	s := &digitalocean.Spec{}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "primary.class") {
		t.Errorf("expected primary.class error, got %v", err)
	}
}

func TestSpec_Validate_rejectsReplicaWithoutClass(t *testing.T) {
	t.Parallel()

	s := &digitalocean.Spec{
		Primary:  digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
		Replicas: &digitalocean.Replicas{Num: 1},
	}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "replicas.class") {
		t.Errorf("expected replicas.class error, got %v", err)
	}
}
