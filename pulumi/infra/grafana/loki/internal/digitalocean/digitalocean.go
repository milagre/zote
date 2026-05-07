// Package digitalocean creates the DigitalOcean Spaces resources Loki needs and renders chart values that consume them.
package digitalocean

import (
	"fmt"

	do "github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/profile"
)

// Args bundles profile, yaml cloud.digitalocean, and placement identity.
type Args struct {
	Profile profile.Profile

	Cloud cloud.Cloud

	// Config is YAML config.cloud.digitalocean.
	Config *Spec

	Namespace string
	Name      string
}

// Setup creates a Pulumi-protected Spaces bucket (durable log store),
// a scoped access key, and a project tag.
func (a *Args) Setup(ctx *pulumi.Context, parent pulumi.Resource) (pulumi.Map, []pulumi.Resource, error) {
	if err := a.validate(); err != nil {
		return nil, nil, err
	}

	objCloud := a.Cloud.DigitalOcean.ForObjectStorage()

	var region pulumi.StringInput
	if a.Config.Region != "" {
		region = pulumi.String(a.Config.Region)
	} else {
		vpc := do.LookupVpcOutput(ctx, do.LookupVpcOutputArgs{
			Id: objCloud.VPCID().ToStringOutput().ToStringPtrOutput(),
		})
		region = vpc.Region()
	}

	resourceName := fmt.Sprintf("%s-%s", a.Namespace, a.Name)

	bucket, err := do.NewSpacesBucket(ctx, resourceName, &do.SpacesBucketArgs{
		Name:   pulumi.String(resourceName),
		Region: region.ToStringOutput().ToStringPtrOutput(),
		Acl:    pulumi.String("private").ToStringPtrOutput(),
	}, pulumi.Parent(parent), pulumi.Protect(true))
	if err != nil {
		return nil, nil, fmt.Errorf("spaces bucket: %w", err)
	}

	if _, err := do.NewProjectResources(ctx, resourceName+"-project", &do.ProjectResourcesArgs{
		Project: objCloud.ProjectID().ToStringOutput(),
		Resources: pulumi.StringArray{
			bucket.BucketUrn,
		},
	}, pulumi.Parent(parent)); err != nil {
		return nil, nil, fmt.Errorf("project resources: %w", err)
	}

	key, err := do.NewSpacesKey(ctx, resourceName, &do.SpacesKeyArgs{
		Name: pulumi.String(resourceName),
		Grants: do.SpacesKeyGrantArray{
			&do.SpacesKeyGrantArgs{
				Bucket:     bucket.Name,
				Permission: pulumi.String("readwrite"),
			},
		},
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, nil, fmt.Errorf("spaces key: %w", err)
	}

	return valuesFor(a, bucket, key, region), []pulumi.Resource{bucket, key}, nil
}

func (a *Args) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Config == nil {
		return fmt.Errorf("cloud.digitalocean config is required")
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("digitalocean config: %w", err)
	}
	if a.Cloud.DigitalOcean == nil {
		return fmt.Errorf("Cloud.DigitalOcean is required")
	}
	objCloud := a.Cloud.DigitalOcean.ForObjectStorage()
	if objCloud.VPCID() == nil {
		return fmt.Errorf("Cloud.DigitalOcean ForObjectStorage VPCID is nil")
	}
	if objCloud.ProjectID() == nil {
		return fmt.Errorf("Cloud.DigitalOcean ForObjectStorage ProjectID is nil")
	}

	return nil
}

// valuesFor renders SimpleScalable + S3-compatible storage.
// SingleBinary replicas are zeroed so the chart doesn't stand up a
// parallel StatefulSet alongside the scalable workloads.
func valuesFor(
	a *Args,
	bucket *do.SpacesBucket,
	key *do.SpacesKey,
	region pulumi.StringInput,
) pulumi.Map {
	bucketName := bucket.Name
	endpoint := pulumi.Sprintf("%s.digitaloceanspaces.com", region)

	resources := pulumi.Map{
		"requests": pulumi.Map{
			"cpu":    pulumi.String(a.Profile.MinCoresMilli()),
			"memory": pulumi.String(a.Profile.MinMemMiB()),
		},
		"limits": pulumi.Map{
			"cpu":    pulumi.String(a.Profile.MaxCoresMilli()),
			"memory": pulumi.String(a.Profile.MaxMemMiB()),
		},
	}

	return pulumi.Map{
		"deploymentMode": pulumi.String("SimpleScalable"),
		"loki": pulumi.Map{
			"storage": pulumi.Map{
				"type": pulumi.String("s3"),
				"bucketNames": pulumi.Map{
					"chunks": bucketName,
					"ruler":  bucketName,
					"admin":  bucketName,
				},
				"s3": pulumi.Map{
					"endpoint":         endpoint,
					"region":           region,
					"accessKeyId":      key.AccessKey,
					"secretAccessKey":  key.SecretKey,
					"s3ForcePathStyle": pulumi.Bool(false),
					"insecure":         pulumi.Bool(false),
				},
			},
			"schemaConfig": pulumi.Map{
				"configs": pulumi.Array{
					pulumi.Map{
						"from":         pulumi.String("2024-04-01"),
						"store":        pulumi.String("tsdb"),
						"object_store": pulumi.String("s3"),
						"schema":       pulumi.String("v13"),
						"index": pulumi.Map{
							"prefix": pulumi.String("index_"),
							"period": pulumi.String("24h"),
						},
					},
				},
			},
		},
		"write": pulumi.Map{
			"replicas":  pulumi.Int(2),
			"resources": resources,
		},
		"read": pulumi.Map{
			"replicas":  pulumi.Int(2),
			"resources": resources,
		},
		"backend": pulumi.Map{
			"replicas":  pulumi.Int(2),
			"resources": resources,
		},
		"singleBinary": pulumi.Map{
			"replicas": pulumi.Int(0),
		},
		"gateway":      pulumi.Map{"enabled": pulumi.Bool(true)},
		"chunksCache":  pulumi.Map{"enabled": pulumi.Bool(false)},
		"resultsCache": pulumi.Map{"enabled": pulumi.Bool(false)},
		"test":         pulumi.Map{"enabled": pulumi.Bool(false)},
		"lokiCanary":   pulumi.Map{"enabled": pulumi.Bool(false)},
		"monitoring": pulumi.Map{
			"selfMonitoring": pulumi.Map{"enabled": pulumi.Bool(false)},
		},
	}
}
