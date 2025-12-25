package zstatsd

import (
	"fmt"

	"github.com/milagre/zote/go/zcmd"
	"github.com/smira/go-statsd"
)

var _ zcmd.Aspect = Aspect{}

type Aspect struct {
}

func (a Aspect) Apply(c zcmd.Configurable) {
	c.AddString("statsd-addr")
	c.AddString("statsd-tag-style")
}

func (a Aspect) Client(env zcmd.Env) (*statsd.Client, error) {
	addr := env.String("statsd-addr")
	tagStyle := env.String("statsd-tag-style")

	tagFormat := statsd.TagFormatGraphite
	switch tagStyle {
	case "datadog":
		tagFormat = statsd.TagFormatDatadog
	case "graphite":
		tagFormat = statsd.TagFormatGraphite
	case "influxdb":
		tagFormat = statsd.TagFormatInfluxDB
	default:
		return nil, fmt.Errorf("invalid statsd tag style: %s", tagStyle)
	}

	return statsd.NewClient(addr, statsd.TagStyle(tagFormat)), nil
}
