package mysql_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/svc/mysql"
	"github.com/milagre/zote/pulumi/svc/mysql/internal/container"
	"github.com/milagre/zote/pulumi/svc/mysql/internal/digitalocean"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validRawProfile() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func validContainer() *container.Spec {
	return &container.Spec{
		Primary: container.Tier{Profile: validRawProfile()},
		Replica: container.Tier{Profile: validRawProfile()},
	}
}

func validDigitalOcean() *mysql.Cloud {
	return &mysql.Cloud{
		DigitalOcean: &digitalocean.Spec{
			Primary: digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
		},
	}
}

func TestConfig_Validate_acceptsContainer(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Version: "8.0", Container: validContainer()}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_acceptsCloudDigitalOcean(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Version: "8", Cloud: validDigitalOcean()}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_rejectsMissingVersion(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Container: validContainer()}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}

func TestConfig_Validate_rejectsBothBackends(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Version: "8", Container: validContainer(), Cloud: validDigitalOcean()}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected mutual-exclusion error, got %v", err)
	}
}

func TestConfig_Validate_rejectsNeitherBackend(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Version: "8"}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "backend is required") {
		t.Errorf("expected backend-required error, got %v", err)
	}
}

func TestConfig_Validate_rejectsEmptyCloud(t *testing.T) {
	t.Parallel()

	c := mysql.Config{Version: "8", Cloud: &mysql.Cloud{}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "at least one provider") {
		t.Errorf("expected provider-required error, got %v", err)
	}
}

func TestConfig_Validate_rejectsEmptyDigitalOceanPrimary(t *testing.T) {
	t.Parallel()

	c := mysql.Config{
		Version: "8",
		Cloud: &mysql.Cloud{
			DigitalOcean: &digitalocean.Spec{},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "primary.class") {
		t.Errorf("expected primary.class error, got %v", err)
	}
}

func TestConfig_Validate_rejectsReplicaWithoutClass(t *testing.T) {
	t.Parallel()

	c := mysql.Config{
		Version: "8",
		Cloud: &mysql.Cloud{
			DigitalOcean: &digitalocean.Spec{
				Primary:  digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
				Replicas: &digitalocean.Replicas{Num: 1},
			},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "replicas.class") {
		t.Errorf("expected replicas.class error, got %v", err)
	}
}

func TestConfig_Validate_acceptsZeroReplicasWithoutClass(t *testing.T) {
	t.Parallel()

	c := mysql.Config{
		Version: "8",
		Cloud: &mysql.Cloud{
			DigitalOcean: &digitalocean.Spec{
				Primary:  digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
				Replicas: &digitalocean.Replicas{Num: 0},
			},
		},
	}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_rejectsInvalidContainerProfile(t *testing.T) {
	t.Parallel()

	c := mysql.Config{
		Version: "8",
		Container: &container.Spec{
			Primary: container.Tier{},
			Replica: container.Tier{Profile: validRawProfile()},
		},
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "primary.profile") {
		t.Errorf("expected primary.profile error, got %v", err)
	}
}
