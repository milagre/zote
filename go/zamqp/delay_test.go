package zamqp

import (
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestAdjustedDelay(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		delay    time.Duration
		expected time.Duration
	}{
		{65201 * time.Millisecond, 65 * time.Second},
		{9612 * time.Millisecond, 10 * time.Second},
		{5 * time.Second, 5 * time.Second},
		{2500 * time.Millisecond, 2500 * time.Millisecond},
		{1 * time.Second, 1 * time.Second},
		{123 * time.Millisecond, 120 * time.Millisecond},
		{12 * time.Millisecond, 12 * time.Millisecond},
		{1 * time.Millisecond, 1 * time.Millisecond},
		{0 * time.Second, 0 * time.Second},
	} {
		t.Run(test.delay.String(), func(t *testing.T) {
			require.Equal(t, test.expected, adjustedDelay(test.delay))
		})
	}
}

func TestDelayName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "delay-debounce-router-6s", delayName("debounce-router", 5500*time.Millisecond))
}

func TestDelayQueue(t *testing.T) {
	t.Parallel()

	queue := delayQueue(Exchange{Name: "debounce-router", Type: ExchangeTypeDirect}, 30*time.Second)

	require.Equal(t, "delay-debounce-router-30s", queue.Name)
	require.Equal(t, 30000, queue.Options[amqp091.QueueMessageTTLArg])
	require.Equal(t, "debounce-router", queue.Options["x-dead-letter-exchange"])
	_, hasDLRK := queue.Options["x-dead-letter-routing-key"]
	require.False(t, hasDLRK)
	require.Equal(t, amqp091.QueueTypeQuorum, queue.Options[amqp091.QueueTypeArg])
	require.Equal(t, "at-least-once", queue.Options["x-dead-letter-strategy"])
}

func TestDelayDeclarations(t *testing.T) {
	t.Parallel()

	name := delayName("debounce-router", 30*time.Second)
	decl := delayDeclarations(
		Exchange{Name: "debounce-router", Type: ExchangeTypeDirect},
		30*time.Second,
	)

	require.Len(t, decl.Exchanges, 1)
	require.Equal(t, name, decl.Exchanges[0].Name)
	require.Equal(t, string(ExchangeTypeDirect), string(decl.Exchanges[0].Type))
	require.Len(t, decl.Queues, 1)
	require.Equal(t, name, decl.Queues[0].Name)
	require.Len(t, decl.Bindings, 1)
	require.Equal(t, name, decl.Bindings[0].ExchangeName)
	require.Equal(t, name, decl.Bindings[0].QueueName)
}
