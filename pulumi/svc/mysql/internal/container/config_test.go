package container_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/svc/mysql/internal/container"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validRaw() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func TestSpec_Validate_acceptsPrimaryAndReplica(t *testing.T) {
	t.Parallel()

	s := &container.Spec{
		Primary: container.Tier{Profile: validRaw()},
		Replica: container.Tier{Profile: validRaw()},
	}
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

func TestSpec_Validate_rejectsInvalidPrimaryProfile(t *testing.T) {
	t.Parallel()

	s := &container.Spec{
		Replica: container.Tier{Profile: validRaw()},
	}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "primary.profile") {
		t.Errorf("expected primary.profile error, got %v", err)
	}
}
