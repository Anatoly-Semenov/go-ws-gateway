package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/pkg/metrics"
	"github.com/anatoly-dev/go-ws-gateway/pkg/models"
	"github.com/anatoly-dev/go-ws-gateway/pkg/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Manager struct {
	clients       map[string]*Client
	mutex         sync.RWMutex
	redisManager  *redis.ConnectionManager
	logger        *zap.Logger
	instanceID    string
	upgrader      websocket.Upgrader
	messageBuffer chan *MessageEnvelope
	metrics       *metrics.WebSocketMetrics
}

type MessageEnvelope struct {
	UserID  string
	Message *models.Message
}

type Client struct {
	ID         string
	UserID     string
	Connection *websocket.Conn
	Send       chan []byte
	Manager    *Manager
	LastPingAt time.Time
	Connected  time.Time
	mu         sync.Mutex
}

func NewManager(redisManager *redis.ConnectionManager, logger *zap.Logger, instanceID string) *Manager {
	manager := &Manager{
		clients:       make(map[string]*Client),
		redisManager:  redisManager,
		logger:        logger,
		instanceID:    instanceID,
		messageBuffer: make(chan *MessageEnvelope, 1000),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	go manager.processMessages()

	return manager
}

func (m *Manager) SetMetrics(metrics *metrics.WebSocketMetrics) {
	m.metrics = metrics
}

func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))

		if m.metrics != nil {
			m.metrics.AuthenticationErrorCount.Inc()
		}

		return
	}

	clientID := uuid.New().String()
	client := &Client{
		ID:         clientID,
		UserID:     userID,
		Connection: conn,
		Send:       make(chan []byte, 256),
		Manager:    m,
		LastPingAt: time.Now(),
		Connected:  time.Now(),
	}

	m.mutex.Lock()
	m.clients[clientID] = client

	if m.metrics != nil {
		m.metrics.ActiveConnections.Set(float64(len(m.clients)))
		m.metrics.ConnectionsTotal.Inc()
	}

	m.mutex.Unlock()

	redisClient := models.NewClient(clientID, userID)
	redisClient.SetMetadata("instance_id", m.instanceID)
	redisClient.SetMetadata("connected_at", client.Connected.Unix())

	ctx := context.Background()
	if err := m.redisManager.RegisterClient(ctx, redisClient); err != nil {
		m.logger.Error("Failed to register client with Redis", zap.Error(err), zap.String("clientID", clientID))
	}

	go client.writePump()
	go client.readPump()
}

func (m *Manager) SendToUser(userID string, message *models.Message) {
	startTime := time.Now()

	m.messageBuffer <- &MessageEnvelope{
		UserID:  userID,
		Message: message,
	}

	if m.metrics != nil {
		m.metrics.MessageBufferSize.Set(float64(len(m.messageBuffer)))

		if len(m.messageBuffer) > cap(m.messageBuffer)*90/100 {
			m.metrics.MessageBufferOverflow.Inc()
		}

		m.metrics.MessagesSent.WithLabelValues(string(message.Type)).Inc()

		m.metrics.MessageLatency.Observe(time.Since(startTime).Seconds())
	}
}

func (m *Manager) processMessages() {
	for envelope := range m.messageBuffer {
		localDelivered := m.deliverToLocalClients(envelope.UserID, envelope.Message)

		if !localDelivered {
			ctx := context.Background()
			if err := m.redisManager.PublishToUser(ctx, envelope.UserID, envelope.Message); err != nil {
				m.logger.Error("Failed to publish message to Redis",
					zap.Error(err),
					zap.String("userID", envelope.UserID))
			}
		}
	}
}

func (m *Manager) deliverToLocalClients(userID string, message *models.Message) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	delivered := false
	msgBytes, err := json.Marshal(message)
	if err != nil {
		m.logger.Error("Failed to marshal message", zap.Error(err))
		return false
	}

	for _, client := range m.clients {
		if client.UserID == userID {
			select {
			case client.Send <- msgBytes:
				delivered = true
			default:
			}
		}
	}

	return delivered
}

func (m *Manager) removeClient(client *Client) {
	m.mutex.Lock()
	delete(m.clients, client.ID)

	if m.metrics != nil {
		m.metrics.ActiveConnections.Set(float64(len(m.clients)))

		connectionDuration := time.Since(client.Connected).Seconds()
		m.metrics.ConnectionDuration.Observe(connectionDuration)
	}

	m.mutex.Unlock()

	ctx := context.Background()
	if err := m.redisManager.UnregisterClient(ctx, client.ID, client.UserID); err != nil {
		m.logger.Error("Failed to unregister client from Redis",
			zap.Error(err),
			zap.String("clientID", client.ID))
	}

	close(client.Send)
}

func (m *Manager) GetClients() []*Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	return clients
}

func (m *Manager) GetClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}

func (m *Manager) Close(ctx context.Context) error {
	m.logger.Info("Closing WebSocket manager")

	m.mutex.Lock()
	for _, client := range m.clients {
		client.Connection.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server shutting down"))
		close(client.Send)
	}
	m.clients = make(map[string]*Client)
	m.mutex.Unlock()

	close(m.messageBuffer)

	m.logger.Info("WebSocket manager closed")
	return nil
}

func (c *Client) readPump() {
	defer func() {
		c.Manager.removeClient(c)
		c.Connection.Close()
	}()

	c.Connection.SetReadLimit(4096)
	c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Connection.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.LastPingAt = time.Now()
		c.mu.Unlock()
		c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				c.Manager.logger.Info("WebSocket closed unexpectedly",
					zap.Error(err),
					zap.String("clientID", c.ID))

				if c.Manager.metrics != nil {
					c.Manager.metrics.UnexpectedCloseCount.Inc()
				}
			}
			break
		}

		c.Manager.logger.Debug("Received message from client",
			zap.String("clientID", c.ID),
			zap.ByteString("message", message))

		if c.Manager.metrics != nil {
			c.Manager.metrics.BytesReceived.Add(float64(len(message)))
			c.Manager.metrics.MessagesReceived.WithLabelValues("client_message").Inc()
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Connection.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			if c.Manager.metrics != nil {
				c.Manager.metrics.BytesSent.Add(float64(len(message)))
			}

			n := len(c.Send)
			for i := 0; i < n; i++ {
				additionalMsg := <-c.Send
				w.Write(additionalMsg)

				if c.Manager.metrics != nil {
					c.Manager.metrics.BytesSent.Add(float64(len(additionalMsg)))
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
