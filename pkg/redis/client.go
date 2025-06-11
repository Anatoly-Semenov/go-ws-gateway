package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/pkg/config"
	"github.com/anatoly-dev/go-ws-gateway/pkg/models"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type ConnectionManager struct {
	client     *redis.Client
	pubsub     *redis.PubSub
	logger     *zap.Logger
	instanceID string
}

const (
	clientPrefix     = "ws:client:"
	userClientsKey   = "ws:user:%s:clients"
	channelPrefix    = "ws:channel:"
	broadcastChannel = "ws:broadcast"
)

func NewConnectionManager(cfg *config.RedisConfig, logger *zap.Logger, instanceID string) (*ConnectionManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	pubsub := client.Subscribe(ctx, broadcastChannel, channelPrefix+instanceID)

	manager := &ConnectionManager{
		client:     client,
		pubsub:     pubsub,
		logger:     logger,
		instanceID: instanceID,
	}

	go manager.subscribe()

	return manager, nil
}

func (m *ConnectionManager) RegisterClient(ctx context.Context, client *models.Client) error {
	clientData, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client data: %w", err)
	}

	clientKey := clientPrefix + client.ID
	userClientsKey := fmt.Sprintf(userClientsKey, client.UserID)

	pipe := m.client.Pipeline()
	pipe.Set(ctx, clientKey, clientData, 24*time.Hour)
	pipe.SAdd(ctx, userClientsKey, client.ID)
	pipe.Expire(ctx, userClientsKey, 24*time.Hour)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to register client in Redis: %w", err)
	}

	return nil
}

func (m *ConnectionManager) UnregisterClient(ctx context.Context, clientID string, userID string) error {
	clientKey := clientPrefix + clientID
	userClientsKey := fmt.Sprintf(userClientsKey, userID)

	pipe := m.client.Pipeline()
	pipe.Del(ctx, clientKey)
	pipe.SRem(ctx, userClientsKey, clientID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unregister client from Redis: %w", err)
	}

	return nil
}

func (m *ConnectionManager) GetUserClients(ctx context.Context, userID string) ([]*models.Client, error) {
	userClientsKey := fmt.Sprintf(userClientsKey, userID)

	clientIDs, err := m.client.SMembers(ctx, userClientsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get client IDs for user: %w", err)
	}

	if len(clientIDs) == 0 {
		return []*models.Client{}, nil
	}

	var keys []string
	for _, id := range clientIDs {
		keys = append(keys, clientPrefix+id)
	}

	values, err := m.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get client data: %w", err)
	}

	var clients []*models.Client
	for _, val := range values {
		if val == nil {
			continue
		}

		strVal, ok := val.(string)
		if !ok {
			m.logger.Warn("invalid client data type in Redis", zap.Any("value", val))
			continue
		}

		var client models.Client
		if err := json.Unmarshal([]byte(strVal), &client); err != nil {
			m.logger.Warn("failed to unmarshal client data", zap.Error(err))
			continue
		}

		clients = append(clients, &client)
	}

	return clients, nil
}

func (m *ConnectionManager) PublishToUser(ctx context.Context, userID string, message *models.Message) error {
	userClientsKey := fmt.Sprintf(userClientsKey, userID)

	clientIDs, err := m.client.SMembers(ctx, userClientsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get client IDs for user: %w", err)
	}

	if len(clientIDs) == 0 {
		return nil
	}

	msgData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var instanceChannels []string
	instanceMap := make(map[string]bool)

	for _, clientID := range clientIDs {
		clientKey := clientPrefix + clientID
		clientData, err := m.client.Get(ctx, clientKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			m.logger.Warn("failed to get client data", zap.Error(err), zap.String("clientID", clientID))
			continue
		}

		var client models.Client
		if err := json.Unmarshal([]byte(clientData), &client); err != nil {
			m.logger.Warn("failed to unmarshal client data", zap.Error(err))
			continue
		}

		instanceID, ok := client.GetMetadata("instance_id")
		if !ok {
			continue
		}

		instanceIDStr, ok := instanceID.(string)
		if !ok {
			continue
		}

		if !instanceMap[instanceIDStr] {
			instanceMap[instanceIDStr] = true
			instanceChannels = append(instanceChannels, channelPrefix+instanceIDStr)
		}
	}

	for _, channel := range instanceChannels {
		pubMsg := struct {
			UserID  string          `json:"user_id"`
			Message json.RawMessage `json:"message"`
		}{
			UserID:  userID,
			Message: msgData,
		}

		pubData, err := json.Marshal(pubMsg)
		if err != nil {
			m.logger.Error("failed to marshal pub message", zap.Error(err))
			continue
		}

		if err := m.client.Publish(ctx, channel, pubData).Err(); err != nil {
			m.logger.Error("failed to publish message", zap.Error(err), zap.String("channel", channel))
		}
	}

	return nil
}

func (m *ConnectionManager) Close() error {
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	m.logger.Info("Closing Redis connection manager")

	var pubsubErr, clientErr error
	
	if m.pubsub != nil {
		pubsubErr = m.pubsub.Close()
		if pubsubErr != nil {
			m.logger.Error("Error closing Redis pubsub", zap.Error(pubsubErr))
		}
	}

	if m.client != nil {
		clientErr = m.client.Close()
		if clientErr != nil {
			m.logger.Error("Error closing Redis client", zap.Error(clientErr))
		}
	}

	if pubsubErr != nil {
		return pubsubErr
	}

	return clientErr
}

func (m *ConnectionManager) subscribe() {
	ctx := context.Background()
	ch := m.pubsub.Channel()

	for msg := range ch {
		m.handleRedisMessage(ctx, msg)
	}
}

func (m *ConnectionManager) handleRedisMessage(ctx context.Context, msg *redis.Message) {
	m.logger.Debug("received Redis message", zap.String("channel", msg.Channel), zap.String("payload", msg.Payload))
}
