package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/anatoly-dev/go-ws-gateway/internal/service"
	"github.com/anatoly-dev/go-ws-gateway/pkg/config"
	"github.com/anatoly-dev/go-ws-gateway/pkg/handlers"
	"github.com/anatoly-dev/go-ws-gateway/pkg/kafka"
	"github.com/anatoly-dev/go-ws-gateway/pkg/metrics"
	"github.com/anatoly-dev/go-ws-gateway/pkg/redis"
	"github.com/anatoly-dev/go-ws-gateway/pkg/websocket"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type Application struct {
	configPath     string
	cfg            *config.Config
	logger         *zap.Logger
	instanceID     string
	redisManager   *redis.ConnectionManager
	wsManager      *websocket.Manager
	kafkaConsumer  *kafka.Consumer
	messageService *service.MessageService
	wsHandler      *handlers.WebSocketHandler
	healthHandler  *handlers.HealthCheckHandler
	metrics        *metrics.Metrics
	metricsHandler *metrics.MetricsHandler
	server         *service.Server
}

func NewApplication(configPath string) *Application {
	return &Application{
		configPath: configPath,
		instanceID: uuid.New().String(),
	}
}

func (a *Application) Init() error {
	if err := a.initConfig(); err != nil {
		return err
	}

	if err := a.initLogger(); err != nil {
		return err
	}

	a.logger.Info("Starting WebSocket Gateway",
		zap.String("instanceID", a.instanceID),
		zap.String("version", "1.0.0"))

	a.initMetrics()

	if err := a.initRedis(); err != nil {
		return err
	}

	a.initWebsocket()

	if err := a.initKafka(); err != nil {
		return err
	}

	a.initServices()
	a.initHandlers()
	a.initServer()

	return nil
}

func (a *Application) initConfig() error {
	cfg, err := config.LoadConfig(a.configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg
	return nil
}

func (a *Application) initLogger() error {
	logger, err := config.NewLogger(&a.cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	a.logger = logger
	return nil
}

func (a *Application) initMetrics() {
	if !a.cfg.Metrics.Enabled {
		a.logger.Info("Metrics collection is disabled")
		return
	}

	a.logger.Info("Initializing metrics",
		zap.String("namespace", a.cfg.Metrics.Namespace),
		zap.String("path", a.cfg.Metrics.Path))

	a.metrics = metrics.NewMetrics(a.cfg.Metrics.Namespace)
	a.metricsHandler = metrics.NewMetricsHandler(a.metrics, a.logger)
}

func (a *Application) initRedis() error {
	redisManager, err := redis.NewConnectionManager(&a.cfg.Redis, a.logger, a.instanceID)
	if err != nil {
		return fmt.Errorf("failed to create Redis connection manager: %w", err)
	}
	a.redisManager = redisManager
	return nil
}

func (a *Application) initWebsocket() {
	a.wsManager = websocket.NewManager(a.redisManager, a.logger, a.instanceID)

	if a.metrics != nil {
		a.wsManager.SetMetrics(&a.metrics.WebSocket)
	}
}

func (a *Application) initKafka() error {
	kafkaConsumer, err := kafka.NewConsumer(&a.cfg.Kafka, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	a.kafkaConsumer = kafkaConsumer

	if a.metrics != nil {
		a.kafkaConsumer.SetMetrics(&a.metrics.Kafka)
	}

	return nil
}

func (a *Application) initServices() {
	a.messageService = service.NewMessageService(a.kafkaConsumer, a.wsManager, a.logger)

	if a.metrics != nil {
		a.messageService.SetMetrics(&a.metrics.Business)
	}
}

func (a *Application) initHandlers() {
	a.wsHandler = handlers.NewWebSocketHandler(a.wsManager, a.logger)
	a.healthHandler = handlers.NewHealthCheckHandler(a.wsManager, a.logger)
}

func (a *Application) initServer() {
	a.server = service.NewServer(
		a.wsHandler,
		a.healthHandler,
		a.messageService,
		a.logger,
		&a.cfg.Server,
		a.metricsHandler,
		&a.cfg.Metrics,
	)
}

func (a *Application) Run() error {
	return a.server.Start()
}

func (a *Application) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a.logger.Info("Stopping application")

	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			a.logger.Error("Error shutting down server", zap.Error(err))
		}
	}

	if a.kafkaConsumer != nil {
		a.kafkaConsumer.Stop()
	}

	if a.redisManager != nil {
		if err := a.redisManager.Close(); err != nil {
			a.logger.Error("Error closing Redis connection", zap.Error(err))
		}
	}

	if a.logger != nil {
		a.logger.Sync()
	}

	a.logger.Info("Application stopped")
}

func NewServeCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the WebSocket Gateway server",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := NewApplication(configPath)
			if err := app.Init(); err != nil {
				return err
			}
			defer app.Stop()
			return app.Run()
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")

	return cmd
}
