package zstats

type Tags map[string]string

type Stats interface {
	AddTag(key string, value string) Stats
	AddTags(Tags Tags) Stats
	AddPrefix(prefix string) Stats
	WithTag(key string, value string) Stats
	WithTags(Tags Tags) Stats
	WithPrefix(prefix string) Stats

	Count(name string, value float64)
	Gauge(name string, value float64)
	Timer(name string, cb func())

	// Histo(name string, value int64)
}

type Adapter interface {
	Count(name string, value float64, tags Tags)
	Gauge(name string, value float64, tags Tags)
	Timer(name string, cb func(), tags Tags)

	// Histo(name string, value int64, tags Tags)
}
