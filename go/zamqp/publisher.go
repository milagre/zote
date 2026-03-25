package zamqp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"

	"github.com/milagre/zote/go/ztrace"
)

type Publisher interface {
	Publish(ctx context.Context, msg Message) (PublishResult, error)
}

type PublishResult struct {
	JobID     string
	MessageID string
}

func NewPublisherFromDetails(details ConnectionDetails) Publisher {
	return dsnPublisher{
		details: details,
	}
}

type dsnPublisher struct {
	details ConnectionDetails
}

func (p dsnPublisher) Publish(ctx context.Context, msg Message) (PublishResult, error) {
	conn, err := Dial(p.details)
	if err != nil {
		return PublishResult{}, fmt.Errorf("dialing for publisher connection: %w", err)
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

func (p connPublisher) Publish(ctx context.Context, msg Message) (PublishResult, error) {
	ch, err := p.conn.NewChannel()
	if err != nil {
		return PublishResult{}, fmt.Errorf("opening channel: %w", err)
	}
	defer ch.Close()

	exchange := msg.Exchange()
	if !msg.Options().SkipExchangeDeclaration && exchange.Name != AnonymousExchange.Name {
		err = ExecuteDeclarations(ch, Declarations{Exchanges: []Exchange{exchange}})
		if err != nil {
			return PublishResult{}, fmt.Errorf("declaring exchange: %w", err)
		}
	}

	confs := ch.channel.NotifyPublish(make(chan amqp091.Confirmation, 1))
	err = ch.channel.Confirm(false)
	if err != nil {
		return PublishResult{}, fmt.Errorf("enabling confirm mode: %w", err)
	}

	publishing, err := messageToPublishing(msg)
	if err != nil {
		return PublishResult{}, fmt.Errorf("preparing publishing: %w", err)
	}

	mergedHeaders := mergePublishHeaders(ctx, msg)
	publishing.Headers = mergedHeaders.toTable()

	err = ch.channel.PublishWithContext(ctx, exchange.Name, msg.Options().RoutingKey, true, false, publishing)
	if err != nil {
		return PublishResult{}, fmt.Errorf("publishing: %w", err)
	}

	select {
	case <-ctx.Done():
		return PublishResult{}, fmt.Errorf("context canceled while publishing: %w", ctx.Err())
	case c := <-confs:
		if !c.Ack {
			return PublishResult{}, fmt.Errorf("amqp server rejected publish")
		}
	}

	return PublishResult{
		JobID:     mergedHeaders[headerJobID].(string),
		MessageID: mergedHeaders[headerMessageID].(string),
	}, nil
}

func mergePublishHeaders(ctx context.Context, msg Message) Headers {
	mo := msg.Options()
	h := Headers{}
	for k, v := range mo.Headers {
		if k == headerJobID {
			continue
		}
		h[k] = v
	}

	if mo.JobID != "" {
		h[headerJobID] = mo.JobID
	} else {
		h[headerJobID] = uuid.NewString()
	}

	h[headerMessageID] = uuid.NewString()

	if _, ok := h[headerCorrelationID]; !ok {
		if t, ok := ztrace.ID(ctx); ok {
			h[headerCorrelationID] = t
		} else {
			h[headerCorrelationID] = ztrace.NewID()
		}
	}

	return h
}
