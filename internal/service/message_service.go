package service

import (
	"github.com/anatoly-dev/go-ws-gateway/pkg/kafka"
	"github.com/anatoly-dev/go-ws-gateway/pkg/models"
	"github.com/anatoly-dev/go-ws-gateway/pkg/websocket"
	"go.uber.org/zap"
)

type MessageService struct {
	kafkaConsumer *kafka.Consumer
	wsManager     *websocket.Manager
	logger        *zap.Logger
}

func NewMessageService(
	kafkaConsumer *kafka.Consumer,
	wsManager *websocket.Manager,
	logger *zap.Logger,
) *MessageService {
	service := &MessageService{
		kafkaConsumer: kafkaConsumer,
		wsManager:     wsManager,
		logger:        logger,
	}

	service.registerMessageHandlers()

	return service
}

func (s *MessageService) registerMessageHandlers() {
	s.kafkaConsumer.RegisterHandler(models.MessageTypeBalanceUpdate, func(msg *models.Message) error {
		s.logger.Info("Handling balance update message",
			zap.String("messageID", msg.ID),
			zap.String("userID", msg.UserID))

		s.wsManager.SendToUser(msg.UserID, msg)
		return nil
	})

	s.kafkaConsumer.RegisterHandler(models.MessageTypeUserBlock, func(msg *models.Message) error {
		s.logger.Info("Handling user block message",
			zap.String("messageID", msg.ID),
			zap.String("userID", msg.UserID))

		s.wsManager.SendToUser(msg.UserID, msg)
		return nil
	})

	s.kafkaConsumer.RegisterHandler(models.MessageTypeNotification, func(msg *models.Message) error {
		s.logger.Info("Handling notification message",
			zap.String("messageID", msg.ID),
			zap.String("userID", msg.UserID))

		s.wsManager.SendToUser(msg.UserID, msg)
		return nil
	})
}

func (s *MessageService) Start() error {
	s.logger.Info("Starting message service")
	return s.kafkaConsumer.Start()
}

func (s *MessageService) Stop() {
	s.logger.Info("Stopping message service")
	s.kafkaConsumer.Stop()
}
