package zstatsd

import (
	"time"

	"github.com/smira/go-statsd"

	"github.com/milagre/zote/go/zstats"
)

var _ zstats.Adapter = &adapter{}

type adapter struct {
	client *statsd.Client
}

func (a *adapter) tags(tags zstats.Tags) []statsd.Tag {
	result := []statsd.Tag{}
	for k, v := range tags {
		result = append(result, statsd.StringTag(k, v))
	}
	return result
}

func (a *adapter) Count(name string, value float64, tags zstats.Tags) {
	a.client.FIncr(name, value, a.tags(tags)...)
}

func (a *adapter) Gauge(name string, value float64, tags zstats.Tags) {
	a.client.FGauge(name, value, a.tags(tags)...)
}

func (a *adapter) Timer(name string, cb func(), tags zstats.Tags) {
	start := time.Now()
	defer func() {
		a.client.Timing(name, time.Since(start).Nanoseconds(), a.tags(tags)...)
	}()
	cb()
}

func NewAdapter(client *statsd.Client) zstats.Adapter {
	return &adapter{
		client: client,
	}
}
