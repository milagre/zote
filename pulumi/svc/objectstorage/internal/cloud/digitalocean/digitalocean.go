// Package digitalocean provisions a DigitalOcean Spaces bucket and access key.
package digitalocean

import (
	"fmt"

	do "github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/types"
)

type Args struct {
	Env   env.Env
	Cloud cloud.Cloud

	Prefix    string
	Namespace string
	Name      string

	Config *Config
}

func Setup(ctx *pulumi.Context, parent pulumi.Resource, a *Args) (*types.Result, error) {
	if a == nil {
		return nil, fmt.Errorf("args is required")
	}
	if a.Namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}
	if a.Name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	if a.Config == nil {
		return nil, fmt.Errorf("Config is required")
	}
	if err := a.Config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if a.Cloud.DigitalOcean == nil {
		return nil, fmt.Errorf("Cloud.DigitalOcean is required")
	}

	cfg := a.Config

	objCloud := a.Cloud.DigitalOcean.ForObjectStorage()

	bucketNames := make([]string, len(cfg.Buckets))
	for i, b := range cfg.Buckets {
		bucketNames[i] = b.Name
	}

	var region pulumi.StringInput
	if cfg.Region != "" {
		region = pulumi.String(cfg.Region)
	} else {
		vpc := do.LookupVpcOutput(ctx, do.LookupVpcOutputArgs{
			Id: objCloud.VPCID().ToStringOutput().ToStringPtrOutput(),
		})
		region = vpc.Region()
	}

	resourceName := fmt.Sprintf("%s-%s-%s-%s", a.Env.AppName, a.Env.ID(), a.Namespace, a.Name)
	finalBuckets := make(map[string]string, len(bucketNames))
	buckets := make([]*do.SpacesBucket, 0, len(bucketNames))
	grants := make(do.SpacesKeyGrantArray, 0, len(bucketNames))
	projectUrns := make(pulumi.StringArray, 0, len(bucketNames))

	for _, b := range bucketNames {
		bucketName := fmt.Sprintf("%s-%s", resourceName, b)
		finalBuckets[b] = bucketName
		bucket, err := do.NewSpacesBucket(ctx, bucketName, &do.SpacesBucketArgs{
			Name:   pulumi.String(bucketName),
			Region: region.ToStringOutput().ToStringPtrOutput(),
			Acl:    pulumi.String("private").ToStringPtrOutput(),
		}, pulumi.Parent(parent), pulumi.Protect(true))
		if err != nil {
			return nil, fmt.Errorf("spaces bucket %q: %w", b, err)
		}

		buckets = append(buckets, bucket)
		projectUrns = append(projectUrns, bucket.BucketUrn)
		grants = append(grants, &do.SpacesKeyGrantArgs{Bucket: bucket.Name, Permission: pulumi.String("readwrite")})
	}

	if _, err := do.NewProjectResources(ctx, resourceName+"-project", &do.ProjectResourcesArgs{
		Project:   objCloud.ProjectID().ToStringOutput(),
		Resources: projectUrns,
	}, pulumi.Parent(parent)); err != nil {
		return nil, fmt.Errorf("project resources: %w", err)
	}

	key, err := do.NewSpacesKey(ctx, resourceName, &do.SpacesKeyArgs{
		Name:   pulumi.String(resourceName),
		Grants: grants,
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("spaces key: %w", err)
	}

	endpointHost := pulumi.Sprintf("%s.digitaloceanspaces.com", region)
	endpointURL := pulumi.Sprintf("https://%s", endpointHost)

	return &types.Result{
		S3: types.Endpoint{
			URL:  endpointURL,
			Host: endpointHost,
			Port: pulumi.String("443").ToStringOutput(),
		},
		Creds: types.Credentials{AccessKey: key.AccessKey, SecretKey: key.SecretKey},
		// Spaces is always TLS; not "insecure".
		Insecure: pulumi.Bool(false).ToBoolOutput(),
		Buckets:  finalBuckets,
		Deps: func() []pulumi.Resource {
			deps := []pulumi.Resource{key}
			for _, b := range buckets {
				deps = append(deps, b)
			}
			return deps
		}(),
	}, nil
}
