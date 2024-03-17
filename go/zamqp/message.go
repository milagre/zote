package zamqp

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

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

func MarshalJsonContent(v any) (data []byte, contentType string, err error) {
	data, err = json.Marshal(v)
	return data, "application/json", err
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
		Compress:   false,
		Headers:    m.finalHeaders(),
		RoutingKey: m.queueDefinition().Name,
	}
}

func (m requeueMessage) Content() ([]byte, string, error) {
	return m.data, m.contentType, nil
}

func (m requeueMessage) Exchange() Exchange {
	return AnonymousExchange
}

func (m requeueMessage) finalHeaders() Headers {
	headers := Headers{}
	for k, v := range m.headers {
		headers[k] = v
	}

	if m.kind == "retry" {
		headers[HeaderAttempt] = attempt(m.headers) + 1
	} else {
		headers[HeaderAttempt] = 1
	}

	return headers
}

func (m *requeueMessage) queueDefinition() Queue {
	if m.queue != nil {
		return *m.queue
	}

	queueName := fmt.Sprintf("%s-%s", m.kind, m.originalQueueName)

	opts := Options{}
	if m.kind == "retry" {
		opts = Options{
			amqp091.QueueMessageTTLArg:  int(m.delay / time.Millisecond),
			amqp091.QueueOverflowArg:    amqp091.QueueOverflowRejectPublish,
			amqp091.QueueTypeArg:        amqp091.QueueTypeQuorum,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": m.originalQueueName,
			"x-dead-letter-strategy":    "at-least-once",
		}
		queueName += "-" + m.delay.String()
	}

	m.queue = &Queue{
		Name:    queueName,
		Options: opts,
	}

	return *m.queue
}
