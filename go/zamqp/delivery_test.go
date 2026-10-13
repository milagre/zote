package zamqp

import (
	"context"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestRequeueOwnKeepsACompressedBodyParsable(t *testing.T) {
	t.Parallel()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)

	requeue, err := deliveryOf(published).requeueOwn("retry", 15*time.Second)
	require.NoError(t, err)

	requeued, err := messageToPublishing(requeue)
	require.NoError(t, err)

	require.Equal(t, "deflate", requeued.ContentEncoding)

	var payload struct {
		AccountID string `json:"account_id"`
	}
	require.NoError(t, deliveryOf(requeued).Parse(&payload))
	require.Equal(t, "137", payload.AccountID)
}

func TestRequeueDataCompressesTheReplacementBodyWhenTheDeliveryArrivedCompressed(t *testing.T) {
	t.Parallel()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)

	requeued, err := messageToPublishing(deliveryOf(published).requeueData(
		"retry", []byte(`{"account_id":"261"}`), "application/json", 0,
	))
	require.NoError(t, err)

	require.Equal(t, "deflate", requeued.ContentEncoding)

	var payload struct {
		AccountID string `json:"account_id"`
	}
	require.NoError(t, deliveryOf(requeued).Parse(&payload))
	require.Equal(t, "261", payload.AccountID)
}

func TestRequeueDataLeavesTheReplacementBodyPlainWhenTheDeliveryArrivedPlain(t *testing.T) {
	t.Parallel()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{},
	))
	require.NoError(t, err)

	requeued, err := messageToPublishing(deliveryOf(published).requeueData(
		"retry", []byte(`{"account_id":"261"}`), "application/json", 0,
	))
	require.NoError(t, err)

	require.Equal(t, "identity", requeued.ContentEncoding)
	require.Equal(t, []byte(`{"account_id":"261"}`), requeued.Body)
}

func TestRequeueOwnRepublishesALegacyCompressBodyAsDeflate(t *testing.T) {
	t.Parallel()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)

	arrived := deliveryOf(published)
	arrived.delivery.ContentEncoding = "compress"

	requeue, err := arrived.requeueOwn("retry", 0)
	require.NoError(t, err)

	requeued, err := messageToPublishing(requeue)
	require.NoError(t, err)

	require.Equal(t, "deflate", requeued.ContentEncoding)

	var payload struct {
		AccountID string `json:"account_id"`
	}
	require.NoError(t, deliveryOf(requeued).Parse(&payload))
	require.Equal(t, "137", payload.AccountID)
}

func TestDecodedBodyDecodesOnce(t *testing.T) {
	t.Parallel()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{Compress: true},
	))
	require.NoError(t, err)

	arrived := deliveryOf(published)

	first, err := arrived.decodedBody()
	require.NoError(t, err)

	arrived.delivery.Body = []byte("not zlib")

	second, err := arrived.decodedBody()
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestRequeueOwnRefusesABodyItCannotDecode(t *testing.T) {
	t.Parallel()

	arrived := deliveryOf(amqp091.Publishing{
		Body:            []byte("not zlib"),
		ContentType:     "application/json",
		ContentEncoding: "compress",
	})

	_, err := arrived.requeueOwn("retry", 0)
	require.Error(t, err)
}

func deliveryOf(publishing amqp091.Publishing) *delivery {
	return &delivery{
		queueName: "myqueue",
		delivery: amqp091.Delivery{
			Body:            publishing.Body,
			ContentType:     publishing.ContentType,
			ContentEncoding: publishing.ContentEncoding,
			Headers:         publishing.Headers,
		},
	}
}

func TestContinuationPublishesUnderTheDeliverysJob(t *testing.T) {
	t.Parallel()

	del := deliveryOf(amqp091.Publishing{Headers: amqp091.Table{headerJobID: "job-1", headerMessageID: "msg-1"}})
	next := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{Compress: true, JobID: "job-other"})

	cont, err := del.continuation(next)
	require.NoError(t, err)

	h := mergePublishHeaders(context.Background(), cont)
	require.Equal(t, "job-1", h[headerJobID])
	require.NotEqual(t, "msg-1", h[headerMessageID])
	require.True(t, cont.Options().Compress)
}

func TestContinuationRefusesADeliveryWithoutAJob(t *testing.T) {
	t.Parallel()

	del := deliveryOf(amqp091.Publishing{})
	next := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{})

	_, err := del.continuation(next)
	require.Error(t, err)
}
