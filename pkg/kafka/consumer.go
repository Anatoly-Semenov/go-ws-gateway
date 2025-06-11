package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/pkg/config"
	"github.com/anatoly-dev/go-ws-gateway/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
)

type MessageHandler func(msg *models.Message) error

type Consumer struct {
	consumer  *kafka.Consumer
	handlers  map[models.MessageType]MessageHandler
	logger    *zap.Logger
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	topics    []string
	isRunning bool
	mutex     sync.Mutex
}

func NewConsumer(cfg *config.KafkaConfig, logger *zap.Logger) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  cfg.BootstrapServers,
		"group.id":           cfg.GroupID,
		"auto.offset.reset":  "latest",
		"enable.auto.commit": true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	consumer := &Consumer{
		consumer: c,
		handlers: make(map[models.MessageType]MessageHandler),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		topics:   cfg.Topics,
	}

	return consumer, nil
}

func (c *Consumer) RegisterHandler(msgType models.MessageType, handler MessageHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[msgType] = handler
}

func (c *Consumer) Start() error {
	c.mutex.Lock()
	if c.isRunning {
		c.mutex.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.isRunning = true
	c.mutex.Unlock()

	if err := c.consumer.SubscribeTopics(c.topics, nil); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	c.wg.Add(1)
	go c.consumeMessages()

	return nil
}

func (c *Consumer) Stop() {
	c.mutex.Lock()
	if !c.isRunning {
		c.mutex.Unlock()
		return
	}
	c.isRunning = false
	c.mutex.Unlock()

	c.logger.Info("Stopping Kafka consumer")

	c.cancel()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("All Kafka consumer routines stopped")
	case <-time.After(10 * time.Second):
		c.logger.Warn("Timeout waiting for Kafka consumer routines to stop")
	}

	if err := c.consumer.Close(); err != nil {
		c.logger.Error("Error closing Kafka consumer", zap.Error(err))
	}

	c.logger.Info("Kafka consumer stopped")
}

func (c *Consumer) consumeMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			ev := c.consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				c.handleMessage(e)
			case kafka.Error:
				c.logger.Error("Kafka error", zap.Error(e), zap.String("code", e.Code().String()))
			default:
			}
		}
	}
}

func (c *Consumer) handleMessage(msg *kafka.Message) {
	c.logger.Debug("Received message from Kafka",
		zap.String("topic", *msg.TopicPartition.Topic),
		zap.Int32("partition", msg.TopicPartition.Partition),
		zap.Int64("offset", int64(msg.TopicPartition.Offset)),
		zap.Time("timestamp", msg.Timestamp),
		zap.ByteString("key", msg.Key),
		zap.Int("value_len", len(msg.Value)))

	var message models.Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.logger.Error("Failed to unmarshal message", zap.Error(err), zap.ByteString("payload", msg.Value))
		return
	}

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	c.mutex.Lock()
	handler, ok := c.handlers[message.Type]
	c.mutex.Unlock()

	if !ok {
		c.logger.Warn("No handler registered for message type", zap.String("type", string(message.Type)))
		return
	}

	if err := handler(&message); err != nil {
		c.logger.Error("Failed to handle message", zap.Error(err), zap.String("type", string(message.Type)))
	}
}
