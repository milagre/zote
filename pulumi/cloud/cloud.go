// Package cloud is the multi-provider container resources receive in
// parallel with their YAML-decoded config. Provider fields are nil
// when that provider isn't configured for the running environment;
// resources read only the field(s) their config selected.
package cloud

import (
	"github.com/milagre/zote/pulumi/cloud/digitalocean"
)

type Cloud struct {
	DigitalOcean *digitalocean.Cloud
}
