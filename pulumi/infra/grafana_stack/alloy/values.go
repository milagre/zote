package alloy

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/util/profile"
)

func chartValues(river pulumi.StringInput, cfg Config) (pulumi.Map, error) {
	alloyBlock := pulumi.Map{
		"configMap": pulumi.Map{
			"content": river,
		},
	}

	if cfg.profileSet() {
		prof, err := profile.New(cfg.Profile)
		if err != nil {
			return nil, fmt.Errorf("profile: %w", err)
		}
		alloyBlock["resources"] = k8sResourcesPulumi(prof)
	}

	return pulumi.Map{"alloy": alloyBlock}, nil
}

func (c Config) profileSet() bool {
	return c.Profile.CPU.Min != "" ||
		c.Profile.CPU.Max != "" ||
		c.Profile.Mem.Min != "" ||
		c.Profile.Mem.Max != ""
}

func k8sResources(prof profile.Profile) map[string]map[string]string {
	return map[string]map[string]string{
		"requests": {
			"cpu":    prof.MinCoresMilli(),
			"memory": prof.MinMemMiB(),
		},
		"limits": {
			"cpu":    prof.MaxCoresMilli(),
			"memory": prof.MaxMemMiB(),
		},
	}
}

func k8sResourcesPulumi(prof profile.Profile) pulumi.Map {
	raw := k8sResources(prof)

	return pulumi.Map{
		"requests": pulumi.Map{
			"cpu":    pulumi.String(raw["requests"]["cpu"]),
			"memory": pulumi.String(raw["requests"]["memory"]),
		},
		"limits": pulumi.Map{
			"cpu":    pulumi.String(raw["limits"]["cpu"]),
			"memory": pulumi.String(raw["limits"]["memory"]),
		},
	}
}
