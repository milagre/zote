package digitalocean

import "fmt"

// Config is YAML under objectstorage.cloud.digitalocean.
type Config struct {
	Region  string   `yaml:"region"`
	Buckets []Bucket `yaml:"buckets"`
}

// Bucket names a Spaces bucket to create. ExpireDays, when positive, deletes
// each object that many days after it was written; zero keeps objects forever.
type Bucket struct {
	Name       string `yaml:"name"`
	ExpireDays int    `yaml:"expire_days,omitempty"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if len(c.Buckets) == 0 {
		return fmt.Errorf("buckets must contain at least one bucket")
	}
	for i, b := range c.Buckets {
		if b.Name == "" {
			return fmt.Errorf("buckets[%d].name is required", i)
		}
		if b.ExpireDays < 0 {
			return fmt.Errorf("buckets[%d].expire_days must not be negative", i)
		}
	}

	return nil
}
