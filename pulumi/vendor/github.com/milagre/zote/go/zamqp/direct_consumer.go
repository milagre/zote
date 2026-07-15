package zamqp

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"

	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zstats"
	"github.com/milagre/zote/go/ztrace"
)

type directConsumer struct {
	conn         Connection
	queueName    string
	concurrency  int
	process      ConsumeFunc
	declarations Declarations
	started      bool
	busyCounter  *atomic.Int64
}

func NewDirectConsumer(conn Connection, declarations Declarations, queueName string, concurrency int, process ConsumeFunc) Consumer {
	return &directConsumer{
		conn:         conn,
		queueName:    queueName,
		declarations: declarations,
		concurrency:  concurrency,
		process:      process,
		started:      false,
		busyCounter:  &atomic.Int64{},
	}
}

func (c *directConsumer) Start(ctx context.Context) error {
	if c.started {
		return errConsumerAlreadyStarted
	}
	c.started = true
	defer func() { c.started = false }()

	processCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerContext := context.WithoutCancel(ctx)

	workerID := 0
	wait := sync.WaitGroup{}
	restart := make(chan int)

	logger := zlog.FromContext(processCtx)
	stats := zstats.FromContext(processCtx)
	stats.AddPrefix(ConsumerStatsPrefix).AddTag("queue", c.queueName)

	consumeChannel, err := makeConsumeChannel(c.conn, c.concurrency)
	if err != nil {
		return err
	}
	defer consumeChannel.Close()

	publisher := NewPublisherFromConnection(c.conn)

	go func() {
		c := make(chan *amqp091.Error)
		consumeChannel.channel.NotifyClose(c)
		err := <-c
		if err != nil {
			logger.Warnf("channel closed unexpectedly: %v", err)
		}
		cancel()
	}()

	err = ExecuteDeclarations(consumeChannel, c.declarations)
	if err != nil {
		return err
	}

	messages := make(chan Delivery, c.concurrency)

	// Stats goroutine
	go (func() {
		interval := 1 * time.Second

		now := time.Now()
		lastInterval := now.Truncate(interval)
		nextInterval := lastInterval.Add(interval)
		initialDelay := nextInterval.Sub(now)

		// Wait for the first tick using time.After
		<-time.After(initialDelay)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stats.Gauge(ConsumerUtilizationMetric, 100*(float64(c.busyCounter.Load())/float64(c.concurrency)))
			case <-processCtx.Done():
				return
			}
		}
	})()

	workerFunc := func(id int) func() {
		return func() {
			consumerLogger := logger.WithField("worker", id)
			consumerContext := zlog.Context(workerContext, consumerLogger)

			defer consumerLogger.Info("Worker shut down")
			consumerLogger.Info("Worker started")

			defer func() {
				if r := recover(); r != nil {
					consumerLogger.Errorf("Direct consumer worker panic: %v; %v", r, string(debug.Stack()))
					restart <- id
				}
			}()

			c.consume(consumerContext, publisher, messages)
		}
	}

	logger.Infof("Starting consume")
	deliveries, err := consumeChannel.channel.ConsumeWithContext(processCtx, c.queueName, "direct", false, false, false, false, amqp091.Table{})
	if err != nil {
		return fmt.Errorf("starting channel consume: %w", err)
	}

	wait.Go(func() {
		defer logger.Info("Dispatcher shut down")
		logger.Info("Dispatcher started")

		for delivery := range deliveries {
			messages <- wrapDelivery(consumeChannel, c.queueName, delivery)
		}

		close(messages)
	})

	wait.Go(func() {
		defer logger.Info("Monitor shut down")
		logger.Info("Monitor started")

		for id := range restart {
			logger.Infof("Restarting worker %d", id)
			wait.Go(workerFunc(id))
		}
	})

	logger.Infof("Spawning %d workers", c.concurrency)
	for i := 0; i < c.concurrency; i++ {
		workerID = workerID + 1
		wait.Go(workerFunc(workerID))
	}

	logger.Infof("Running")

	<-processCtx.Done()

	logger.Info("Shutdown initiated, waiting for work to finish")
	close(restart)

	wait.Wait()

	logger.Info("Shutting down")

	return nil
}

func makeConsumeChannel(conn Connection, concurrency int) (Channel, error) {
	channel, err := conn.NewChannel()
	if err != nil {
		return Channel{}, fmt.Errorf("creating channel: %w", err)
	}

	err = channel.channel.Qos(concurrency, 0, false)
	if err != nil {
		return Channel{}, fmt.Errorf("setting channel qos: %w", err)
	}

	return channel, nil
}

func (c *directConsumer) consume(ctx context.Context, publisher Publisher, deliveries chan Delivery) {
	stats := zstats.FromContext(ctx)
	logger := zlog.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return

		case del := <-deliveries:
			if del == nil {
				logger.Debugf("Channel closed, ending consume loop")
				return
			}

			c.busyCounter.Add(1)
			stats.Count("received", 1)

			h := del.Headers()
			msgCtx := Context(ctx,
				headerStringDefault(h, headerJobID, uuid.NewString),
				headerStringDefault(h, headerMessageID, uuid.NewString),
			)
			msgCtx = ztrace.Context(msgCtx, headerStringDefault(h, headerCorrelationID, ztrace.NewID))

			msgLogger := zlog.FromContext(msgCtx)
			msgLogger = msgLogger.WithFields(zlog.Fields{
				"attempt": del.Attempt(),
				"message": del.Tag(),
			})
			msgCtx = zlog.Context(msgCtx, msgLogger)

			func() {
				defer msgLogger.Info("Complete")
				msgLogger.Info("Starting")

				defer func() {
					if r := recover(); r != nil {
						msgLogger.Info("Panic while processing message, requeuing with delay")
						del.RetryDelayed(ctx, 1*time.Second)
						panic(r)
					}
				}()

				defer stats.Count("completed", 1)
				defer c.busyCounter.Add(-1)

				c.process(msgCtx, publisher, del)
			}()
		}
	}
}
