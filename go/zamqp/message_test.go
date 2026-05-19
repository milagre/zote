package zamqp

import (
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestRequeueMessageQueueDefinitionRetryDelayUsesSecondResolution(t *testing.T) {
	t.Parallel()

	msg := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             5500*time.Millisecond + 123*time.Nanosecond,
		kind:              "retry",
	}

	queue := msg.queueDefinition()

	require.Equal(t, "retry-myqueue-6s", queue.Name)
	require.Equal(t, 6000, queue.Options[amqp091.QueueMessageTTLArg])
}

func TestRequeueMessageQueueDefinitionRetryDelayBucketsFractionalDelays(t *testing.T) {
	t.Parallel()

	first := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             5500*time.Millisecond + 100*time.Nanosecond,
		kind:              "retry",
	}
	second := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             5500*time.Millisecond + 900*time.Nanosecond,
		kind:              "retry",
	}

	require.Equal(t, first.queueDefinition().Name, second.queueDefinition().Name)
}

func TestRequeueMessageQueueDefinitionRetryDelaySubSecondUsesOneSecond(t *testing.T) {
	t.Parallel()

	msg := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             250 * time.Millisecond,
		kind:              "retry",
	}

	queue := msg.queueDefinition()

	require.Equal(t, "retry-myqueue-1s", queue.Name)
	require.Equal(t, 1000, queue.Options[amqp091.QueueMessageTTLArg])
}

func TestRequeueMessageQueueDefinitionRetryDelayZeroUsesZeroSeconds(t *testing.T) {
	t.Parallel()

	msg := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             0,
		kind:              "retry",
	}

	queue := msg.queueDefinition()

	require.Equal(t, "retry-myqueue-0s", queue.Name)
	require.Equal(t, 0, queue.Options[amqp091.QueueMessageTTLArg])
}
