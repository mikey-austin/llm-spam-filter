package logging

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initializes a logger based on configuration
func InitLogger(cfg *config.Config) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.GetString("logging.level") {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var logConfig zap.Config
	if cfg.GetString("logging.format") == "json" {
		logConfig = zap.NewProductionConfig()
	} else {
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logConfig.Level = zap.NewAtomicLevelAt(level)

	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	return logger, nil
}

// InitConsoleLogger initializes a console-friendly logger
func InitConsoleLogger(verbose bool, jsonFormat bool) (*zap.Logger, error) {
	var level zapcore.Level
	if verbose {
		level = zapcore.DebugLevel
	} else {
		level = zapcore.InfoLevel
	}

	var logConfig zap.Config
	if jsonFormat {
		logConfig = zap.NewProductionConfig()
	} else {
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logConfig.Level = zap.NewAtomicLevelAt(level)

	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	return logger, nil
}
