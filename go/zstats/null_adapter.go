package zstats

type nullAdapter struct{}

func (a *nullAdapter) Count(name string, value float64, tags Tags) {}
func (a *nullAdapter) Gauge(name string, value float64, tags Tags) {}
func (a *nullAdapter) Timer(name string, cb func(), tags Tags) {
	cb()
}

// func (a *nullAdapter) Histo(name string, value float64, tags Tags) {}

func NewNullAdapter() Adapter {
	return &nullAdapter{}
}
