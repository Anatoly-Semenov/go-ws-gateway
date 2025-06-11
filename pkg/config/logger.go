package config

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg *LoggerConfig) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	var config zap.Config
	if cfg.Format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)
	config.DisableStacktrace = true

	return config.Build()
}
