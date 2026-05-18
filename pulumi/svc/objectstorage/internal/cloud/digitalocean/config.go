package digitalocean

import "fmt"

// Config is YAML under objectstorage.cloud.digitalocean.
type Config struct {
	Region  string   `yaml:"region"`
	Buckets []Bucket `yaml:"buckets"`
}

// Bucket names a Spaces bucket to create.
type Bucket struct {
	Name string `yaml:"name"`
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
	}

	return nil
}
