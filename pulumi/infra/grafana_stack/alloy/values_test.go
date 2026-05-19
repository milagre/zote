package alloy

import (
	"testing"

	"github.com/milagre/zote/pulumi/util/profile"
)

func TestK8sResources_fromProfile(t *testing.T) {
	t.Parallel()

	prof, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "50m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := k8sResources(prof)
	if got["requests"]["cpu"] != "50m" || got["requests"]["memory"] != "64Mi" {
		t.Errorf("requests: %#v", got["requests"])
	}
	if got["limits"]["cpu"] != "100m" || got["limits"]["memory"] != "128Mi" {
		t.Errorf("limits: %#v", got["limits"])
	}
}
