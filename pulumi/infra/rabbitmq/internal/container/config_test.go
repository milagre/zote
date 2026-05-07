package container_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/infra/rabbitmq/internal/container"
	"github.com/milagre/zote/pulumi/profile"
)

func rawProf() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func TestSpec_Validate_acceptsProfile(t *testing.T) {
	t.Parallel()

	s := &container.Spec{Profile: rawProf()}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestSpec_Validate_rejectsNil(t *testing.T) {
	t.Parallel()

	var s *container.Spec
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil spec error, got %v", err)
	}
}
