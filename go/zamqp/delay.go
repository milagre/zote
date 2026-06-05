package zamqp

import (
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

func adjustedDelay(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}

	if delay >= 5*time.Second {
		return delay.Round(time.Second)
	}

	if delay >= 1*time.Second {
		return delay.Round(100 * time.Millisecond)
	}

	if delay >= 100*time.Millisecond {
		return delay.Round(10 * time.Millisecond)
	}

	return delay.Round(1 * time.Millisecond)
}

func delayName(exchangeName string, delay time.Duration) string {
	return fmt.Sprintf("delay-%s-%s", exchangeName, adjustedDelay(delay))
}

func delayExchange(exchangeName string, delay time.Duration) Exchange {
	return Exchange{
		Name: delayName(exchangeName, delay),
		Type: ExchangeTypeDirect,
	}
}

func delayQueue(exchange Exchange, delay time.Duration) Queue {
	return Queue{
		Name: delayName(exchange.Name, delay),
		Options: Options{
			amqp091.QueueMessageTTLArg: int(adjustedDelay(delay) / time.Millisecond),
			amqp091.QueueOverflowArg:   amqp091.QueueOverflowRejectPublish,
			amqp091.QueueTypeArg:       amqp091.QueueTypeQuorum,
			"x-dead-letter-exchange":   exchange.Name,
			"x-dead-letter-strategy":   "at-least-once",
		},
	}
}

func delayDeclarations(exchange Exchange, delay time.Duration) Declarations {
	name := delayName(exchange.Name, delay)

	return Declarations{
		Exchanges: []Exchange{delayExchange(exchange.Name, delay)},
		Queues:    []Queue{delayQueue(exchange, delay)},
		Bindings: []Binding{
			{
				ExchangeName: name,
				QueueName:    name,
			},
		},
	}
}
