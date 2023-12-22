package zamqp

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/milagre/zote/go/zlog"
	"github.com/rabbitmq/amqp091-go"
)

type ConnectionDetails struct {
	user  string
	pass  string
	host  string
	port  int
	vhost string
}

func NewConnectionDetails(user string, pass string, host string, port int, vhost string) ConnectionDetails {
	return ConnectionDetails{
		user:  user,
		pass:  pass,
		host:  host,
		port:  port,
		vhost: vhost,
	}
}

func (d ConnectionDetails) URI() string {
	return amqp091.URI{
		Scheme:   "amqp",
		Host:     d.host,
		Port:     d.port,
		Username: d.user,
		Password: d.pass,
		Vhost:    d.vhost,
	}.String()
}

func Dial(details ConnectionDetails) (Connection, error) {
	conn, err := amqp091.Dial(details.URI())
	if err != nil {
		return Connection{}, fmt.Errorf("dialing amqp server: %w", err)
	}

	return Connection{
		conn: conn,
	}, nil
}

type Connection struct {
	conn *amqp091.Connection
}

func (c Connection) Close() error {
	return c.conn.Close()
}

func (c Connection) NewChannel() (Channel, error) {
	ac, err := c.conn.Channel()
	if err != nil {
		return Channel{}, fmt.Errorf("opening channel: %w", err)
	}

	channel := Channel{
		conn:    c,
		channel: ac,
	}

	return channel, nil
}

type Channel struct {
	conn    Connection
	channel *amqp091.Channel
	// publishConfirms chan amqp091.Confirmation
}

func (c Channel) Close() error {
	return c.channel.Close()
}

type Queue struct {
	Name string

	NonDurable bool
	AutoDelete bool
	Exclusive  bool
	Options    Options
}

type Exchange struct {
	Name       string
	Type       ExchangeType
	NonDurable bool
	AutoDelete bool
	Internal   bool
	Options    Options
}

type ExchangeType string

const ExchangeTypeDirect = "direct"
const ExchangeTypeFanout = "fanout"
const ExchangeTypeTopic = "topic"
const ExchangeTypeHeaders = "headers"

var AnonymousExchange = Exchange{
	Name: "",
	Type: ExchangeTypeDirect,
}

type Binding struct {
	ExchangeName string
	QueueName    string
	RoutingKey   string
	Options      Options
}

type Options map[string]string

func (o Options) toTable() amqp091.Table {
	table := amqp091.Table{}
	for k, v := range o {
		table[k] = v
	}
	return table
}

type Headers map[string]interface{}

func (h Headers) toTable() amqp091.Table {
	t := amqp091.Table{}
	for k, v := range h {
		t[k] = v
	}
	return t
}

// MessageOptions are optional pieces of data for publishing a message. An empty object is valid.
type MessageOptions struct {
	Compress   bool
	RoutingKey string
	Headers    Headers
}

type Message interface {
	Content() (data []byte, contentType string, err error)
	Options() MessageOptions
	Exchange() Exchange
}

// RawMessage fully implements the Message interface for any data
type RawMessage struct {
	data        []byte
	contentType string
	headers     Headers
	exchange    Exchange
	opts        MessageOptions
}

func NewRawMessage(data []byte, contentType string, exchange Exchange, opts MessageOptions) Message {
	headers := opts.Headers
	if headers == nil {
		headers = Headers{}
	}

	return RawMessage{
		data:        data,
		contentType: contentType,
		exchange:    exchange,
		headers:     headers,
		opts:        opts,
	}
}

func (m RawMessage) Content() (data []byte, contentType string, err error) {
	return m.data, m.contentType, nil
}

func (m RawMessage) Options() MessageOptions {
	return m.opts
}

func (m RawMessage) Exchange() Exchange {
	return m.exchange
}

func (m RawMessage) Headers() Headers {
	return m.headers
}

func messageToPublishing(msg Message) (amqp091.Publishing, error) {
	content, contentType, err := msg.Content()
	if err != nil {
		return amqp091.Publishing{}, fmt.Errorf("getting message content: %w", err)
	}

	body := bytes.NewBuffer(content)

	contentEncoding := "identity"
	if msg.Options().Compress {
		contentEncoding = "compress"

		compressedBody := &bytes.Buffer{}
		w := zlib.NewWriter(compressedBody)
		_, err := io.Copy(w, body)
		if err != nil {
			return amqp091.Publishing{}, fmt.Errorf("preparing message body compression: %w", err)
		}

		err = w.Flush()
		if err != nil {
			return amqp091.Publishing{}, fmt.Errorf("flushing message body compression: %w", err)
		}

		err = w.Close()
		if err != nil {
			return amqp091.Publishing{}, fmt.Errorf("closing messaging body compression: %w", err)
		}

		body = compressedBody
	}

	return amqp091.Publishing{
		Body:            body.Bytes(),
		Headers:         msg.Options().Headers.toTable(),
		ContentType:     contentType,
		ContentEncoding: contentEncoding,
	}, nil
}

type Delivery interface {
	Body() []byte
	Headers() Headers
	ContentType() string
	ContentEncoding() string
	Tag() uint64

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
		return nil
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

type requeueMessage struct {
	data              []byte
	contentType       string
	originalQueueName string
	headers           Headers
	delay             time.Duration
	queue             *Queue
	kind              string
}

var _ Message = requeueMessage{}

func (m requeueMessage) Options() MessageOptions {
	return MessageOptions{
		RoutingKey: m.queueDefinition().Name,
		Compress:   true,
	}
}

func (m requeueMessage) Content() ([]byte, string, error) {
	return m.data, m.contentType, nil
}

func (m requeueMessage) Headers() Headers {
	headers := Headers{}
	for k, v := range m.headers {
		headers[k] = v
	}

	if m.kind == "retry" {
		nextAttempt := 2
		attempt := headers["attempt"]
		if attemptInt, ok := attempt.(int); ok {
			nextAttempt = attemptInt + 1
		}
		headers["attempt"] = strconv.Itoa(nextAttempt)
	} else {
		headers["attempt"] = 1
	}

	return headers
}

func (m requeueMessage) Exchange() Exchange {
	return AnonymousExchange
}

func (m *requeueMessage) queueDefinition() Queue {
	if m.queue != nil {
		return *m.queue
	}

	opts := Options{}
	if m.kind == "retry" {
		opts = Options{
			amqp091.QueueMessageTTLArg:  fmt.Sprintf("%d", m.delay/time.Second),
			amqp091.QueueOverflowArg:    amqp091.QueueOverflowRejectPublish,
			amqp091.QueueTypeArg:        amqp091.QueueTypeQuorum,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": m.originalQueueName,
			"x-dead-letter-strategy":    "at-least-once",
		}
	}

	queueName := fmt.Sprintf("%s-%s-%s", m.kind, m.delay, m.originalQueueName)

	m.queue = &Queue{
		Name:    queueName,
		Options: opts,
	}

	return *m.queue
}
