package zamqp

import (
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestMessageToPublishing_PersistentDeliveryMode(t *testing.T) {
	t.Parallel()

	pub, err := messageToPublishing(NewRawMessage(
		[]byte("payload"),
		"text/plain",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)
	require.Equal(t, amqp091.Persistent, pub.DeliveryMode)
}

func TestMessageToPublishing_TransientDeliveryMode(t *testing.T) {
	t.Parallel()

	pub, err := messageToPublishing(NewRawMessage(
		[]byte("payload"),
		"text/plain",
		Exchange{Name: "dest"},
		MessageOptions{Transient: true},
	))
	require.NoError(t, err)
	require.Equal(t, amqp091.Transient, pub.DeliveryMode)
}

func TestMessageToPublishing_CompressPublishesTheDeflateEncoding(t *testing.T) {
	t.Parallel()

	pub, err := messageToPublishing(NewRawMessage(
		[]byte("payload"),
		"text/plain",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)
	require.Equal(t, "deflate", pub.ContentEncoding)
}

func TestPublishingDeliveryMode(t *testing.T) {
	t.Parallel()

	require.Equal(t, amqp091.Persistent, publishingDeliveryMode(MessageOptions{}))
	require.Equal(t, amqp091.Transient, publishingDeliveryMode(MessageOptions{Transient: true}))
}

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

func TestRequeueMessageQueueDefinitionRetryDelaySubSecond(t *testing.T) {
	t.Parallel()

	msg := &requeueMessage{
		originalQueueName: "myqueue",
		delay:             250 * time.Millisecond,
		kind:              "retry",
	}

	queue := msg.queueDefinition()

	require.Equal(t, "retry-myqueue-250ms", queue.Name)
	require.Equal(t, 250, queue.Options[amqp091.QueueMessageTTLArg])
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
