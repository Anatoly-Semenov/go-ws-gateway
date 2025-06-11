package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/pkg/config"
	"github.com/anatoly-dev/go-ws-gateway/pkg/handlers"
	"go.uber.org/zap"
)

type Server struct {
	server         *http.Server
	wsHandler      *handlers.WebSocketHandler
	healthHandler  *handlers.HealthCheckHandler
	messageService *MessageService
	logger         *zap.Logger
	cfg            *config.ServerConfig
}

func NewServer(
	wsHandler *handlers.WebSocketHandler,
	healthHandler *handlers.HealthCheckHandler,
	messageService *MessageService,
	logger *zap.Logger,
	cfg *config.ServerConfig,
) *Server {
	return &Server{
		wsHandler:      wsHandler,
		healthHandler:  healthHandler,
		messageService: messageService,
		logger:         logger,
		cfg:            cfg,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.wsHandler.HandleConnection)
	mux.HandleFunc("/health", s.healthHandler.HandleHealthCheck)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      mux,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	if err := s.messageService.Start(); err != nil {
		return fmt.Errorf("failed to start message service: %w", err)
	}

	go func() {
		s.logger.Info("Starting server", zap.Int("port", s.cfg.Port))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	return s.waitForShutdown()
}

func (s *Server) waitForShutdown() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	s.logger.Info("Received shutdown signal")

	shutdownTimeout := 30 * time.Second
	if s.cfg.ShutdownTimeout > 0 {
		shutdownTimeout = s.cfg.ShutdownTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	s.logger.Info("Shutting down services", zap.Duration("timeout", shutdownTimeout))

	s.messageService.Stop()

	if err := s.wsHandler.CloseConnections(ctx); err != nil {
		s.logger.Error("Error closing WebSocket connections", zap.Error(err))
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Performing controlled shutdown")

	s.messageService.Stop()

	if err := s.wsHandler.CloseConnections(ctx); err != nil {
		s.logger.Error("Error closing WebSocket connections", zap.Error(err))
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info("Server shutdown completed")
	return nil
}
