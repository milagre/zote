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
	"github.com/milagre/zote/go/zstats"
)

type Delivery interface {
	Body() []byte
	Headers() Headers
	ContentType() string
	ContentEncoding() string
	Tag() uint64
	Attempt() int
	QueueName() string

	Ack(context.Context) error
	Reject(context.Context) error
	Retry(context.Context) error
	RetryWithData(ctx context.Context, content []byte, contentType string) error
	RetryDelayed(context.Context, time.Duration) error
	RetryDelayedWithData(ctx context.Context, delay time.Duration, content []byte, contentType string) error
	Fatal(context.Context) error

	Parse(v interface{}) error
}

type delivery struct {
	channel   Channel
	queueName string
	delivery  amqp091.Delivery
	complete  bool
	lock      sync.Mutex

	decode    sync.Once
	decoded   []byte
	decodeErr error
}

func (m *delivery) Parse(target interface{}) error {
	var parser func([]byte, interface{}) error
	switch m.ContentType() {
	case "application/json":
		parser = json.Unmarshal

	default:
		return fmt.Errorf("unsupported content type '%s', parse the body directly", m.ContentType())
	}

	decoded, err := m.decodedBody()
	if err != nil {
		return fmt.Errorf("decoding message body: %w", err)
	}

	err = parser(decoded, target)
	if err != nil {
		return fmt.Errorf("parsing decoded message body: %w", err)
	}

	return nil
}

// decodedBody is the body decoded from any contentEncoding over the wire.
func (m *delivery) decodedBody() ([]byte, error) {
	m.decode.Do(func() {
		m.decoded, m.decodeErr = decodeBody(m.ContentEncoding(), m.Body())
	})

	return m.decoded, m.decodeErr
}

func decodeBody(contentEncoding string, body []byte) ([]byte, error) {
	var decoder io.Reader
	switch contentEncoding {
	case encodingIdentity, "":
		decoder = bytes.NewBuffer(body)

	case encodingDeflate, encodingCompress:
		r, err := zlib.NewReader(bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("making zlib reader: %w", err)
		}

		decoder = r
	default:
		return nil, fmt.Errorf("unsupported content encoding '%s', parse the body directly", contentEncoding)
	}

	decoded, err := io.ReadAll(decoder)
	if err != nil {
		return nil, fmt.Errorf("reading message body: %w", err)
	}

	return decoded, nil
}

func compressed(contentEncoding string) bool {
	return contentEncoding == encodingDeflate || contentEncoding == encodingCompress
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

func (m *delivery) QueueName() string {
	return m.queueName
}

func (m *delivery) Ack(ctx context.Context) error {
	err := m.respond(func() error {
		return m.delivery.Ack(false)
	})
	if err != nil {
		return fmt.Errorf("acking message: %w", err)
	}

	zstats.FromContext(ctx).Count("delivery.ack", 1)
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

	zstats.FromContext(ctx).Count("delivery.reject", 1)
	zlog.FromContext(ctx).Info("Message rejected")

	return nil
}

// requeueOwn republishes this delivery's body unchanged, but re-encoded
func (m *delivery) requeueOwn(kind string, delay time.Duration) (requeueMessage, error) {
	decoded, err := m.decodedBody()
	if err != nil {
		return requeueMessage{}, fmt.Errorf("decoding message body: %w", err)
	}

	msg := m.requeueData(kind, decoded, m.ContentType(), delay)

	return msg, nil
}

// requeueData republishes a body in place of this delivery's
func (m *delivery) requeueData(kind string, content []byte, contentType string, delay time.Duration) requeueMessage {
	return requeueMessage{
		data:              content,
		contentType:       contentType,
		compress:          compressed(m.ContentEncoding()),
		originalQueueName: m.queueName,
		headers:           m.Headers(),
		delay:             delay,
		kind:              kind,
	}
}

func (m *delivery) Retry(ctx context.Context) error {
	msg, err := m.requeueOwn("retry", 0)
	if err != nil {
		return fmt.Errorf("preparing message for retry: %w", err)
	}

	err = m.respond(func() error {
		return m.requeue(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("retrying message: %w", err)
	}

	zstats.FromContext(ctx).Count("delivery.retry", 1)
	zlog.FromContext(ctx).Info("Message retried")

	return nil
}

func (m *delivery) RetryWithData(ctx context.Context, content []byte, contentType string) error {
	err := m.respond(func() error {
		return m.requeue(ctx, m.requeueData("retry", content, contentType, 0))
	})
	if err != nil {
		return fmt.Errorf("retrying message: %w", err)
	}

	zstats.FromContext(ctx).Count("delivery.retry", 1)
	zlog.FromContext(ctx).Info("Message retried")

	return nil
}

func (m *delivery) RetryDelayed(ctx context.Context, delay time.Duration) error {
	msg, err := m.requeueOwn("retry", delay)
	if err != nil {
		return fmt.Errorf("preparing message for delayed retry: %w", err)
	}

	err = m.respond(func() error {
		return m.requeue(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("retrying message with delay: %w", err)
	}

	zstats.FromContext(ctx).WithTag("delay", delay.String()).Count("delivery.retry", 1)
	zlog.FromContext(ctx).Infof("Message queued for retry in %s", delay)

	return nil
}

func (m *delivery) RetryDelayedWithData(ctx context.Context, delay time.Duration, content []byte, contentType string) error {
	err := m.respond(func() error {
		return m.requeue(ctx, m.requeueData("retry", content, contentType, delay))
	})
	if err != nil {
		return fmt.Errorf("retrying message with delay: %w", err)
	}

	zstats.FromContext(ctx).WithTag("delay", delay.String()).Count("delivery.retry", 1)
	zlog.FromContext(ctx).Infof("Message queued for retry in %s", delay)

	return nil
}

func (m *delivery) Fatal(ctx context.Context) error {
	msg, err := m.requeueOwn("fatal", 0)
	if err != nil {
		return fmt.Errorf("preparing message for the fatal queue: %w", err)
	}

	err = m.respond(func() error {
		return m.requeue(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("retrying message with delay: %w", err)
	}

	zstats.FromContext(ctx).Count("delivery.fatal", 1)
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
	_, err = publisher.Publish(ctx, msg)
	if err != nil {
		return fmt.Errorf("publishing %s: %w", msg.kind, err)
	}

	err = m.delivery.Nack(false, false)
	if err != nil {
		return fmt.Errorf("rejecting message after publishing %s: %w", msg.kind, err)
	}

	return nil
}
