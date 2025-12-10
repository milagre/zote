package zstats

type stats struct {
	adapter Adapter
	prefix  string
	tags    Tags
}

func NewStats(adapter Adapter) *stats {
	return &stats{
		adapter: adapter,
		prefix:  "",
		tags:    Tags{},
	}
}

func (s *stats) WithTag(key string, value string) Stats {
	return s.WithTags(Tags{key: value})
}

func (s *stats) WithTags(Tags Tags) Stats {
	result := NewStats(s.adapter)
	for k, v := range s.tags {
		result.tags[k] = v
	}
	for k, v := range Tags {
		result.tags[k] = v
	}
	return result
}

func (s *stats) AddTag(key string, value string) Stats {
	s.tags[key] = value
	return s
}

func (s *stats) AddTags(Tags Tags) Stats {
	for k, v := range Tags {
		s.tags[k] = v
	}
	return s
}

func (s *stats) AddPrefix(prefix string) Stats {
	if s.prefix == "" {
		s.prefix = prefix
	} else {
		s.prefix = s.prefix + "." + prefix
	}
	return s
}

func (s *stats) WithPrefix(prefix string) Stats {
	result := NewStats(s.adapter)
	result.AddPrefix(prefix)
	return result
}

func (s *stats) Count(name string, value float64) {
	s.adapter.Count(s.prefixedName(name), value, s.tags)
}

func (s *stats) Gauge(name string, value float64) {
	s.adapter.Gauge(s.prefixedName(name), value, s.tags)
}
func (s *stats) Timer(name string, cb func()) { s.adapter.Timer(s.prefixedName(name), cb, s.tags) }

// func (s *stats) Histo(name string, value float64) { s.adapter.Histo(s.prefixedName(name), value, s.tags) }

func (s *stats) prefixedName(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "." + name
}
