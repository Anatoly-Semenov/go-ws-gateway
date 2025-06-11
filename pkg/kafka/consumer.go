package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/pkg/config"
	"github.com/anatoly-dev/go-ws-gateway/pkg/metrics"
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
	metrics   *metrics.KafkaMetrics
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

func (c *Consumer) SetMetrics(metrics *metrics.KafkaMetrics) {
	c.metrics = metrics
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

	lastMessages := make(map[string]time.Time)
	messageCounters := make(map[string]int)
	rateUpdateTicker := time.NewTicker(30 * time.Second)
	defer rateUpdateTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-rateUpdateTicker.C:
			if c.metrics != nil {
				now := time.Now()
				for topic, count := range messageCounters {
					lastTime, ok := lastMessages[topic]
					if ok {
						duration := now.Sub(lastTime).Seconds()
						if duration > 0 {
							rate := float64(count) / duration
							c.metrics.ProcessingRate.WithLabelValues(topic).Set(rate)
						}
					}
					messageCounters[topic] = 0
					lastMessages[topic] = now
				}
			}
		default:
			ev := c.consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				c.handleMessage(e)

				if c.metrics != nil && e.TopicPartition.Topic != nil {
					topic := *e.TopicPartition.Topic

					c.metrics.MessagesProcessed.WithLabelValues(topic).Inc()

					messageCounters[topic]++

					low, high, err := c.consumer.QueryWatermarkOffsets(*e.TopicPartition.Topic, e.TopicPartition.Partition, 5000)
					if err == nil {
						currentLag := high - int64(e.TopicPartition.Offset)
						partitionStr := fmt.Sprintf("%d", e.TopicPartition.Partition)
						c.metrics.ConsumerLag.WithLabelValues(topic, partitionStr).Set(float64(currentLag))

						c.logger.Debug("Kafka consumer lag",
							zap.String("topic", topic),
							zap.String("partition", partitionStr),
							zap.Int64("low", low),
							zap.Int64("high", high),
							zap.Int64("current", int64(e.TopicPartition.Offset)),
							zap.Int64("lag", currentLag))
					}
				}

			case kafka.Error:
				c.logger.Error("Kafka error", zap.Error(e), zap.String("code", e.Code().String()))

				if c.metrics != nil {
					c.metrics.KafkaErrors.WithLabelValues(e.Code().String()).Inc()
				}
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

		if c.metrics != nil {
			c.metrics.DeserializeErrors.Inc()
		}

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
