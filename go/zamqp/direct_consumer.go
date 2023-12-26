package zamqp

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/milagre/zote/go/zlog"
	"github.com/rabbitmq/amqp091-go"
)

type directConsumer struct {
	conn         Connection
	queueName    string
	concurrency  int
	process      ConsumeFunc
	declarations Declarations
	started      bool
}

func NewDirectConsumer(conn Connection, declarations Declarations, queueName string, concurrency int, process ConsumeFunc) Consumer {
	return &directConsumer{
		conn:         conn,
		queueName:    queueName,
		declarations: declarations,
		concurrency:  concurrency,
		process:      process,
		started:      false,
	}
}

func (c *directConsumer) Start(processCtx context.Context, workerContext context.Context) error {
	if c.started {
		return errConsumerAlreadyStarted
	}
	c.started = true
	defer func() { c.started = false }()

	processCtx, cancel := context.WithCancel(processCtx)
	defer cancel()

	workerID := 0
	logger := zlog.FromContext(processCtx)
	wait := sync.WaitGroup{}
	restart := make(chan int)

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

	launch := func(id int) {
		wait.Add(1)
		go func(id int) {
			defer wait.Done()

			consumerLogger := logger.WithField("worker", id)
			consumerContext := zlog.Context(workerContext, consumerLogger)

			defer consumerLogger.Info("Worker shut down")
			consumerLogger.Info("Worker started")

			defer func() {
				if r := recover(); r != nil {
					consumerLogger.Warnf("Direct consumer worker panic: %v; %v", r, string(debug.Stack()))
					restart <- id
				}
			}()

			c.consume(consumerContext, publisher, messages)
		}(id)
	}

	logger.Infof("Starting consume")
	deliveries, err := consumeChannel.channel.ConsumeWithContext(processCtx, c.queueName, "direct", false, false, false, false, amqp091.Table{})
	if err != nil {
		return fmt.Errorf("starting channel consume: %w", err)
	}

	wait.Add(1)
	go func() {
		defer wait.Done()
		defer logger.Info("Dispatcher shut down")
		logger.Info("Dispatcher started")

		for delivery := range deliveries {
			messages <- wrapDelivery(consumeChannel, c.queueName, delivery)
		}

		close(messages)
	}()

	go func() {
		defer logger.Info("Monitor shut down")
		logger.Info("Monitor started")

		for id := range restart {
			logger.Infof("Restarting worker %d", id)
			launch(id)
		}
	}()

	logger.Infof("Spawning %d workers", c.concurrency)
	for i := 0; i < c.concurrency; i++ {
		workerID = workerID + 1
		launch(workerID)
	}

	logger.Infof("Running")

	<-processCtx.Done()

	logger.Info("Shutdown initiated, waiting for work to finish")

	wait.Wait()

	close(restart)

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

			msgLogger := logger.WithFields(zlog.Fields{
				"attempt": del.Attempt(),
				"message": del.Tag(),
			})
			msgCtx := zlog.Context(ctx, msgLogger)

			func() {
				defer msgLogger.Info("Message processed")
				msgLogger.Info("Message received")

				defer func() {
					if r := recover(); r != nil {
						del.RetryDelayed(ctx, 1*time.Second)
						panic(r)
					}
				}()

				c.process(msgCtx, publisher, del)
			}()
		}
	}
}
