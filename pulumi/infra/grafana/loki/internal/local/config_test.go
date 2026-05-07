package local_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/local"
)

func TestSpec_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	s := &local.Spec{}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestSpec_Validate_rejectsNil(t *testing.T) {
	t.Parallel()

	var s *local.Spec
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil spec error, got %v", err)
	}
}
