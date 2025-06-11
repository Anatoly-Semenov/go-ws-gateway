package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anatoly-dev/go-ws-gateway/pkg/websocket"
	"go.uber.org/zap"
)

type WebSocketHandler struct {
	wsManager *websocket.Manager
	logger    *zap.Logger
}

func NewWebSocketHandler(wsManager *websocket.Manager, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
		logger:    logger,
	}
}

func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.logger.Warn("Missing user_id parameter")
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	h.logger.Info("WebSocket connection request", zap.String("userID", userID))
	h.wsManager.HandleConnection(w, r, userID)
}

func (h *WebSocketHandler) CloseConnections(ctx context.Context) error {
	h.logger.Info("Closing all WebSocket connections")
	return h.wsManager.Close(ctx)
}

type HealthCheckHandler struct {
	wsManager *websocket.Manager
	logger    *zap.Logger
}

func NewHealthCheckHandler(wsManager *websocket.Manager, logger *zap.Logger) *HealthCheckHandler {
	return &HealthCheckHandler{
		wsManager: wsManager,
		logger:    logger,
	}
}

func (h *HealthCheckHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	clientCount := h.wsManager.GetClientCount()
	h.logger.Debug("Health check", zap.Int("clientCount", clientCount))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status":"ok","client_count":%d}`, clientCount)))
}
