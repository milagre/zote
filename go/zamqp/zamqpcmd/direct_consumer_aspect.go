package zamqpcmd

import (
	"fmt"

	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zcmd"
)

var _ zcmd.Aspect = DirectConsumerAspect{}

type DirectConsumerAspect struct {
	Aspect
}

func NewDirectConsumerAspect(name string) DirectConsumerAspect {
	return DirectConsumerAspect{
		Aspect: NewAspect(name),
	}
}

func (a DirectConsumerAspect) Apply(c zcmd.Configurable) {
	a.Aspect.Apply(c)
	c.AddInt(a.concurrency())
}

func (a DirectConsumerAspect) Consumer(
	env zcmd.Env,
	declarations zamqp.Declarations,
	queueName string,
	process zamqp.ConsumeFunc,
) (zamqp.Consumer, error) {
	conn, err := a.Aspect.Connection(env)
	if err != nil {
		return nil, fmt.Errorf("getting amqp connection: %w", err)
	}

	consumer := zamqp.NewDirectConsumer(
		conn, declarations,
		queueName,
		env.Int(a.concurrency()),
		process,
	)

	return consumer, nil
}

func (a DirectConsumerAspect) concurrency() string {
	return "concurrency"
}
