package zamqp

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/milagre/zote/go/zlog"
)

type Delivery interface {
	Body() []byte
	Headers() Headers
	ContentType() string
	ContentEncoding() string
	Tag() uint64
	Attempt() int

	Ack(context.Context) error
	Reject(context.Context) error
	Retry(context.Context) error
	RetryDelayed(context.Context, time.Duration) error
	Fatal(context.Context) error

	Parse(v interface{}) error
}

type delivery struct {
	channel   Channel
	queueName string
	delivery  amqp091.Delivery
	complete  bool
	lock      sync.Mutex
}

func (m *delivery) Parse(target interface{}) error {
	var parser func([]byte, interface{}) error
	switch m.ContentType() {
	case "application/json":
		parser = json.Unmarshal

	default:
		return fmt.Errorf("unsupported content type '%s', parse the body directly", m.ContentType())
	}

	var decoder io.Reader
	switch m.ContentEncoding() {
	case "identity", "":
		decoder = bytes.NewBuffer(m.Body())

	case "compress":
		r, err := zlib.NewReader(bytes.NewBuffer(m.Body()))
		if err != nil {
			return fmt.Errorf("making zlib reader: %w", err)
		}

		decoder = r
	default:
		return fmt.Errorf("unsupported content encoding '%s', parse the body directly", m.ContentEncoding())
	}

	decoded, err := io.ReadAll(decoder)
	if err != nil {
		return fmt.Errorf("decoding message body: %w", err)
	}

	err = parser(decoded, target)
	if err != nil {
		return fmt.Errorf("parsing decoded message body: %w", err)
	}

	return nil
}

func wrapDelivery(channel Channel, queueName string, del amqp091.Delivery) Delivery {
	return &delivery{
		channel:   channel,
		queueName: queueName,
		delivery:  del,
		lock:      sync.Mutex{},
		complete:  false,
	}
}

func (m *delivery) respond(cb func() error) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.complete {
		return fmt.Errorf("message already responded to")
	}

	err := cb()
	if err != nil {
		return fmt.Errorf("responding to message: %w", err)
	}

	m.complete = true

	return nil
}

func (m *delivery) Body() []byte {
	return m.delivery.Body
}

func (m *delivery) Headers() Headers {
	return Headers(m.delivery.Headers)
}

func (m *delivery) ContentType() string {
	return m.delivery.ContentType
}

func (m *delivery) ContentEncoding() string {
	return m.delivery.ContentEncoding
}

func (m *delivery) Tag() uint64 {
	return m.delivery.DeliveryTag
}

func (m *delivery) Attempt() int {
	return attempt(m.Headers())
}

func (m *delivery) Ack(ctx context.Context) error {
	err := m.respond(func() error {
		return m.delivery.Ack(false)
	})
	if err != nil {
		return fmt.Errorf("acking message: %w", err)
	}

	zlog.FromContext(ctx).Info("Message acknowledged")

	return nil
}

func (m *delivery) Reject(ctx context.Context) error {
	err := m.respond(func() error {
		return m.delivery.Nack(false, false)
	})
	if err != nil {
		return fmt.Errorf("rejecting message: %w", err)
	}

	zlog.FromContext(ctx).Info("Message rejected")

	return nil
}

func (m *delivery) Retry(ctx context.Context) error {
	// TODO: use `m.requeue` to increment attempt count properly
	err := m.respond(func() error {
		return m.delivery.Nack(false, true)
	})
	if err != nil {
		return fmt.Errorf("retrying message: %w", err)
	}

	zlog.FromContext(ctx).Info("Message retried")

	return nil
}

func (m *delivery) RetryDelayed(ctx context.Context, delay time.Duration) error {
	err := m.respond(func() error {
		return m.requeue(ctx, requeueMessage{
			data:              m.Body(),
			contentType:       m.ContentType(),
			originalQueueName: m.queueName,
			headers:           m.Headers(),
			delay:             delay,
			kind:              "retry",
		})
	})
	if err != nil {
		return fmt.Errorf("retrying message with delay: %w", err)
	}

	zlog.FromContext(ctx).Infof("Message queued for retry in %s", delay)

	return nil
}

func (m *delivery) Fatal(ctx context.Context) error {
	err := m.respond(func() error {
		return m.requeue(ctx, requeueMessage{
			data:              m.Body(),
			contentType:       m.ContentType(),
			originalQueueName: m.queueName,
			headers:           m.Headers(),
			kind:              "fatal",
		})
	})
	if err != nil {
		return fmt.Errorf("retrying message with delay: %w", err)
	}

	zlog.FromContext(ctx).Infof("Message fataled")

	return nil
}

func (m *delivery) requeue(ctx context.Context, msg requeueMessage) error {
	err := ExecuteDeclarations(m.channel, Declarations{
		Queues: []Queue{msg.queueDefinition()},
	})
	if err != nil {
		return fmt.Errorf("executing declarations: %w", err)
	}

	publisher := NewPublisherFromConnection(m.channel.conn)
	err = publisher.Publish(ctx, msg)
	if err != nil {
		return fmt.Errorf("publishing %s: %w", msg.kind, err)
	}

	err = m.delivery.Nack(false, false)
	if err != nil {
		return fmt.Errorf("rejecting message after publishing %s: %w", msg.kind, err)
	}

	return nil
}
