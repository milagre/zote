package zamqpcmd

import (
	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zcmd"
)

var _ zcmd.Aspect = PublisherAspect{}

type PublisherAspect struct {
	Aspect
}

func NewPublisherAspect(name string) PublisherAspect {
	return PublisherAspect{
		Aspect: NewAspect(name),
	}
}

func (a PublisherAspect) Apply(c zcmd.Configurable) {
	a.Aspect.Apply(c)
}

func (a PublisherAspect) Publisher(
	env zcmd.Env,
) zamqp.Publisher {
	return zamqp.NewPublisherFromDetails(a.Aspect.Details(env))
}
