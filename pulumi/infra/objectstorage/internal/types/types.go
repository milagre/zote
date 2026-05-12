package types

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type Endpoint struct {
	URL  pulumi.StringOutput
	Host pulumi.StringOutput
	Port pulumi.StringOutput
}

type Credentials struct {
	AccessKey pulumi.StringOutput
	SecretKey pulumi.StringOutput
}

type Result struct {
	S3       Endpoint
	Creds    Credentials
	Insecure pulumi.BoolOutput
	Deps     []pulumi.Resource

	// Buckets maps each bucket name from objectstorage config to the provisioned name in the backend
	Buckets map[string]string
}
