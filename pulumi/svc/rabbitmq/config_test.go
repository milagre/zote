package rabbitmq_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/svc/rabbitmq"
	"github.com/milagre/zote/pulumi/svc/rabbitmq/internal/container"
	"github.com/milagre/zote/pulumi/util/profile"
)

func rawProf() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func validRabbitConfig() rabbitmq.Config {
	return rabbitmq.Config{
		Version: "3.12",
		Container: &container.Spec{
			Profile: rawProf(),
		},
	}
}

func TestConfig_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	c := validRabbitConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestConfig_Validate_rejectsMissingVersion(t *testing.T) {
	t.Parallel()

	c := validRabbitConfig()
	c.Version = ""

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}
