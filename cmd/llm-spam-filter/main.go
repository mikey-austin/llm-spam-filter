package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/adapters/bedrock"
	"github.com/mikey/llm-spam-filter/internal/adapters/cache"
	"github.com/mikey/llm-spam-filter/internal/adapters/filter"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/ports"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	// Initialize AWS client                                                        16:08:39 [382/1876]
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.GetString("bedrock.region")),
	)
	if err != nil {
		logger.Fatal("Failed to load AWS configuration", zap.Error(err))
	}

	// Initialize Bedrock client
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	llmClient := bedrock.NewBedrockClient(
		bedrockClient,
		cfg.GetString("bedrock.model_id"),
		cfg.GetInt("bedrock.max_tokens"),
		float32(cfg.GetFloat64("bedrock.temperature")),
		float32(cfg.GetFloat64("bedrock.top_p")),
		cfg.GetInt("bedrock.max_body_size"),
		logger,
	)

	// Initialize cache
	var cacheRepo core.CacheRepository
	cacheTTL, err := time.ParseDuration(cfg.GetString("cache.ttl"))
	if err != nil {
		logger.Fatal("Invalid cache TTL", zap.Error(err))
	}
	cleanupFreq, err := time.ParseDuration(cfg.GetString("cache.cleanup_frequency"))
	if err != nil {
		logger.Fatal("Invalid cleanup frequency", zap.Error(err))
	}

	switch cfg.GetString("cache.type") {
	case "memory":
		cacheRepo = cache.NewMemoryCache(logger, cleanupFreq)
	case "sqlite":
		sqlitePath := cfg.GetString("cache.sqlite_path")
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
			logger.Fatal("Failed to create SQLite directory", zap.Error(err))
		}
		cacheRepo, err = cache.NewSQLiteCache(sqlitePath, logger, cleanupFreq)
		if err != nil {
			logger.Fatal("Failed to initialize SQLite cache", zap.Error(err))
		}
	case "mysql":
		mysqlDSN := cfg.GetString("cache.mysql_dsn")
		cacheRepo, err = cache.NewMySQLCache(mysqlDSN, logger, cleanupFreq)
		if err != nil {
			logger.Fatal("Failed to initialize MySQL cache", zap.Error(err))
		}
	default:
		logger.Fatal("Invalid cache type", zap.String("type", cfg.GetString("cache.type")))
	}

	// Get whitelisted domains
	whitelistedDomains := cfg.GetStringSlice("spam.whitelisted_domains")
	if len(whitelistedDomains) > 0 {
		logger.Info("Loaded whitelisted domains", zap.Strings("domains", whitelistedDomains))
	}

	// Initialize spam filter service
	spamService := core.NewSpamFilterService(
		llmClient,
		cacheRepo,
		logger,
		cfg.GetBool("cache.enabled"),
		cacheTTL,
		cfg.GetFloat64("spam.threshold"),
		whitelistedDomains,
	)

	// Initialize filter
	var emailFilter ports.EmailFilter
	switch cfg.GetString("server.filter_type") {
	case "postfix":
		emailFilter = filter.NewPostfixFilter(
			spamService,
			logger,
			cfg.GetString("server.listen_address"),
			cfg.GetBool("server.block_spam"),
			cfg.GetString("server.headers.spam"),
			cfg.GetString("server.headers.score"),
			cfg.GetString("server.headers.reason"),
		)
	case "milter":
		emailFilter = filter.NewMilterFilter(
			spamService,
			logger,
			cfg.GetString("server.listen_address"),
			cfg.GetBool("server.block_spam"),
			cfg.GetString("server.headers.spam"),
			cfg.GetString("server.headers.score"),
			cfg.GetString("server.headers.reason"),
		)
	default:
		logger.Fatal("Invalid filter type", zap.String("type", cfg.GetString("server.filter_type")))
	}

	// Start the filter
	if err := emailFilter.Start(); err != nil {
		logger.Fatal("Failed to start filter", zap.Error(err))
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Info("Shutting down...")

	// Stop the filter
	if err := emailFilter.Stop(); err != nil {
		logger.Error("Failed to stop filter", zap.Error(err))
	}

	// Stop the cache if needed
	if memCache, ok := cacheRepo.(*cache.MemoryCache); ok {
		memCache.Stop()
	} else if sqliteCache, ok := cacheRepo.(*cache.SQLiteCache); ok {
		sqliteCache.Stop()
	} else if mysqlCache, ok := cacheRepo.(*cache.MySQLCache); ok {
		mysqlCache.Stop()
	}

	logger.Info("Shutdown complete")
}

func loadConfig() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/llm-spam-filter/")
	v.AddConfigPath("$HOME/.llm-spam-filter")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// Set defaults
	v.SetDefault("server.filter_type", "postfix")
	v.SetDefault("server.listen_address", "0.0.0.0:10025")
	v.SetDefault("server.block_spam", false)
	v.SetDefault("server.headers.spam", "X-Spam-Status")
	v.SetDefault("server.headers.score", "X-Spam-Score")
	v.SetDefault("server.headers.reason", "X-Spam-Reason")
	v.SetDefault("bedrock.region", "us-east-1")
	v.SetDefault("bedrock.model_id", "anthropic.claude-v2")
	v.SetDefault("bedrock.max_tokens", 1000)
	v.SetDefault("bedrock.temperature", 0.1)
	v.SetDefault("bedrock.top_p", 0.9)
	v.SetDefault("bedrock.max_body_size", 4096)
	v.SetDefault("spam.threshold", 0.7)
	v.SetDefault("spam.whitelisted_domains", []string{})
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl", "24h")
	v.SetDefault("cache.cleanup_frequency", "1h")
	v.SetDefault("cache.sqlite_path", "/data/spam_cache.db")
	v.SetDefault("cache.mysql_dsn", "user:password@tcp(localhost:3306)/spam_filter")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("SPAM_FILTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, using defaults
	}

	return v, nil
}

func initLogger(cfg *viper.Viper) (*zap.Logger, error) {
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

	var config zap.Config
	if cfg.GetString("logging.format") == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}
	config.Level = zap.NewAtomicLevelAt(level)

	return config.Build()
}
