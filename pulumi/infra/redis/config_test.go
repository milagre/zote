package redis_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/infra/redis"
	rediscloud "github.com/milagre/zote/pulumi/infra/redis/internal/cloud"
	"github.com/milagre/zote/pulumi/infra/redis/internal/cloud/digitalocean"
	"github.com/milagre/zote/pulumi/infra/redis/internal/container"
	"github.com/milagre/zote/pulumi/profile"
)

func rawProf() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func validRedisConfig() redis.Config {
	return redis.Config{
		Version:  "7",
		Shards:   1,
		Replicas: 0,
		Container: &container.Spec{
			Profile: rawProf(),
		},
	}
}

func TestConfig_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	c := validRedisConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestConfig_Validate_rejectsMissingVersion(t *testing.T) {
	t.Parallel()

	c := validRedisConfig()
	c.Version = ""

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}

func TestConfig_Validate_acceptsStandardOmittedShardsReplicas(t *testing.T) {
	t.Parallel()

	c := validRedisConfig()
	c.Shards = 0
	c.Replicas = 0

	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestConfig_Validate_rejectsClusterWithoutShards(t *testing.T) {
	t.Parallel()

	c := validRedisConfig()
	c.Shards = 0
	c.Replicas = 1

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "shards") {
		t.Errorf("expected shards error, got %v", err)
	}
}

func TestConfig_Validate_requiresExactlyOneBackend_bothSet(t *testing.T) {
	t.Parallel()

	c := validRedisConfig()
	c.Cloud = &rediscloud.Config{DigitalOcean: &digitalocean.Config{}}

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("expected exactly one backend error, got %v", err)
	}
}

func TestConfig_Validate_requiresExactlyOneBackend_neitherSet(t *testing.T) {
	t.Parallel()

	c := redis.Config{}

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("expected exactly one backend error, got %v", err)
	}
}

func TestConfig_Validate_cloudRequiresDigitalOcean(t *testing.T) {
	t.Parallel()

	c := redis.Config{
		Cloud: &rediscloud.Config{},
	}

	if err := c.Validate(); err == nil || !strings.Contains(strings.ToLower(err.Error()), "digitalocean") {
		t.Errorf("expected digitalocean error, got %v", err)
	}
}

func TestConfig_Validate_acceptsCloudMinimal(t *testing.T) {
	t.Parallel()

	c := redis.Config{
		Cloud: &rediscloud.Config{
			DigitalOcean: &digitalocean.Config{},
		},
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
