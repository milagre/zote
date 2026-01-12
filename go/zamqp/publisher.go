package zamqp

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

func NewPublisherFromDetails(details ConnectionDetails) Publisher {
	return dsnPublisher{
		details: details,
	}
}

type dsnPublisher struct {
	details ConnectionDetails
}

func (p dsnPublisher) Publish(ctx context.Context, msg Message) error {
	conn, err := Dial(p.details)
	if err != nil {
		return fmt.Errorf("dialing for publisher connection: %w", err)
	}
	defer conn.Close()

	return NewPublisherFromConnection(conn).Publish(ctx, msg)
}

func NewPublisherFromConnection(conn Connection) Publisher {
	return connPublisher{
		conn: conn,
	}
}

type connPublisher struct {
	conn Connection
}

func (p connPublisher) Publish(ctx context.Context, msg Message) error {
	ch, err := p.conn.NewChannel()
	if err != nil {
		return fmt.Errorf("opening channel: %w", err)
	}
	defer ch.Close()

	exchange := msg.Exchange()
	if !msg.Options().SkipExchangeDeclaration && exchange.Name != AnonymousExchange.Name {
		err = ExecuteDeclarations(ch, Declarations{Exchanges: []Exchange{exchange}})
		if err != nil {
			return fmt.Errorf("declaring exchange: %w", err)
		}
	}

	confs := ch.channel.NotifyPublish(make(chan amqp091.Confirmation, 1))
	err = ch.channel.Confirm(false)
	if err != nil {
		return fmt.Errorf("enabling confirm mode: %w", err)
	}

	publishing, err := messageToPublishing(msg)
	if err != nil {
		return fmt.Errorf("preparing publishing: %w", err)
	}

	for k, v := range msg.Options().Headers {
		publishing.Headers[k] = v
	}

	err = ch.channel.PublishWithContext(ctx, exchange.Name, msg.Options().RoutingKey, true, false, publishing)
	if err != nil {
		return fmt.Errorf("publishing: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled while publishing: %w", ctx.Err())
	case c := <-confs:
		if !c.Ack {
			return fmt.Errorf("amqp server rejected publish")
		}
	}

	return nil
}
