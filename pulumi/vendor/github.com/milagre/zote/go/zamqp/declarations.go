package zamqp

import (
	"fmt"
)

type Declarations struct {
	Queues    []Queue
	Exchanges []Exchange
	Bindings  []Binding
}

func ExecuteDeclarations(channel Channel, declarations Declarations) error {
	for _, e := range declarations.Exchanges {
		err := channel.channel.ExchangeDeclare(
			e.Name,
			string(e.Type),
			!e.NonDurable,
			e.AutoDelete,
			e.Internal,
			false,
			e.Options.toTable(),
		)
		if err != nil {
			return fmt.Errorf("exchange declaration for '%s' failed: %w", e.Name, err)
		}
	}

	for _, q := range declarations.Queues {
		_, err := channel.channel.QueueDeclare(
			q.Name,
			!q.NonDurable,
			q.AutoDelete,
			q.Exclusive,
			false,
			q.Options.toTable(),
		)
		if err != nil {
			return fmt.Errorf("queue declaration for '%s' failed: %w", q.Name, err)
		}
	}

	for _, b := range declarations.Bindings {
		err := channel.channel.QueueBind(
			b.QueueName,
			b.RoutingKey,
			b.ExchangeName,
			false,
			b.Options.toTable(),
		)
		if err != nil {
			return fmt.Errorf("binding declaration for '%s'->'%s' failed: %w", b.ExchangeName, b.QueueName, err)
		}
	}

	return nil
}
