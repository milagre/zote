package zinfluxdb

import (
	"fmt"
	"net/url"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zcmd/zaspect"
)

var _ zcmd.Aspect = Aspect{}

type Aspect struct {
	name string
}

func NewAspect(name string) Aspect {
	return Aspect{
		name: name,
	}
}

func (a Aspect) Apply(c zcmd.Configurable) {
	c.AddString(a.scheme()).Default("http")
	c.AddString(a.host()).Default("localhost")
	c.AddInt(a.port()).Default(80)
	c.AddString(a.org())
	c.AddString(a.bucket())
	c.AddString(a.user())
	c.AddString(a.token())
}

type Client struct {
	influxdb2.Client

	org    string
	bucket string
}

func (c Client) DefaultOrg() string {
	return c.org
}

func (c Client) DefaultBucket() string {
	return c.bucket
}

func (a Aspect) Client(env zcmd.Env, options *influxdb2.Options) Client {
	uri := url.URL{
		Scheme: env.String(a.scheme()),
		Host:   fmt.Sprintf("%s:%d", env.String(a.host()), env.Int(a.port())),
	}

	token := env.String(a.token())
	org := env.String(a.org())
	bucket := env.String(a.bucket())

	opts := options
	if opts == nil {
		opts = influxdb2.DefaultOptions()
	}

	client := Client{
		Client: influxdb2.NewClientWithOptions(uri.String(), token, opts),
		org:    org,
		bucket: bucket,
	}

	return client
}

// Option constructors

func (a Aspect) scheme() string {
	return a.opt("scheme")
}

func (a Aspect) host() string {
	return a.opt("host")
}

func (a Aspect) port() string {
	return a.opt("port")
}

func (a Aspect) org() string {
	return a.opt("org")
}

func (a Aspect) bucket() string {
	return a.opt("bucket")
}

func (a Aspect) user() string {
	return a.opt("user")
}

func (a Aspect) token() string {
	return a.opt("token")
}

func (a Aspect) opt(opt string) string {
	return zaspect.Format("influxdb-%s-%s", a.name, opt)
}
