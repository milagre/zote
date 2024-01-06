package zamqp

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cast"
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
	Name string
	Type ExchangeType

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

type Options map[string]interface{}

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

const HeaderAttempt = "x-zote-attempt"

func attempt(headers Headers) int {
	v, ok := headers[HeaderAttempt]
	if !ok {
		return 1
	}

	i, err := cast.ToIntE(v)
	if err != nil {
		return 1
	}

	return i
}
