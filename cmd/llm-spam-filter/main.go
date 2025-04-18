package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/factory"
	"github.com/mikey/llm-spam-filter/internal/logging"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.InitLogger(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize LLM client
	llmFactory := factory.NewLLMFactory(cfg, logger)
	llmClient, err := llmFactory.CreateLLMClient()
	if err != nil {
		logger.Fatal("Failed to create LLM client", zap.Error(err))
	}

	// Initialize cache
	cacheFactory := factory.NewCacheFactory(cfg, logger)
	cacheRepo, err := cacheFactory.CreateCacheRepository()
	if err != nil {
		logger.Fatal("Failed to create cache repository", zap.Error(err))
	}
	
	cacheTTL, err := cacheFactory.GetCacheTTL()
	if err != nil {
		logger.Fatal("Invalid cache TTL", zap.Error(err))
	}
	
	cacheEnabled := cacheFactory.IsCacheEnabled()

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
		cacheEnabled,
		cacheTTL,
		cfg.GetFloat64("spam.threshold"),
		whitelistedDomains,
	)

	// Initialize filter
	filterFactory := factory.NewFilterFactory(cfg, logger, spamService)
	emailFilter, err := filterFactory.CreateEmailFilter()
	if err != nil {
		logger.Fatal("Failed to create email filter", zap.Error(err))
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

	// Close any resources that need closing
	if closer, ok := llmClient.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Failed to close LLM client", zap.Error(err))
		}
	}

	// Stop the cache if needed
	if stopper, ok := cacheRepo.(interface{ Stop() }); ok {
		stopper.Stop()
	}

	logger.Info("Shutdown complete")
}
