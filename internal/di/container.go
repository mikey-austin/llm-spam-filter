package di

import (
	"time"

	"go.uber.org/dig"
	"go.uber.org/zap"

	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/factory"
	"github.com/mikey/llm-spam-filter/internal/logging"
	"github.com/mikey/llm-spam-filter/internal/ports"
	"github.com/mikey/llm-spam-filter/internal/utils"
)

// BuildContainer creates and configures a dependency injection container
func BuildContainer() (*dig.Container, error) {
	container := dig.New()

	// Register configuration
	if err := container.Provide(config.New); err != nil {
		return nil, err
	}

	// Register logger
	if err := container.Provide(logging.InitLogger); err != nil {
		return nil, err
	}

	// Register factories
	if err := container.Provide(factory.NewLLMFactory); err != nil {
		return nil, err
	}
	if err := container.Provide(factory.NewCacheFactory); err != nil {
		return nil, err
	}
	if err := container.Provide(factory.NewFilterFactory); err != nil {
		return nil, err
	}
	if err := container.Provide(factory.NewTextProcessorFactory); err != nil {
		return nil, err
	}

	// Register LLM client
	if err := container.Provide(func(f *factory.LLMFactory) (core.LLMClient, error) {
		return f.CreateLLMClient()
	}); err != nil {
		return nil, err
	}

	// Register cache repository
	if err := container.Provide(func(f *factory.CacheFactory) (core.CacheRepository, error) {
		return f.CreateCacheRepository()
	}); err != nil {
		return nil, err
	}

	// Register cache TTL and enabled flag
	if err := container.Provide(func(f *factory.CacheFactory) (time.Duration, error) {
		return f.GetCacheTTL()
	}); err != nil {
		return nil, err
	}
	if err := container.Provide(func(f *factory.CacheFactory) bool {
		return f.IsCacheEnabled()
	}); err != nil {
		return nil, err
	}

	// Register whitelisted domains
	if err := container.Provide(func(cfg *config.Config, logger *zap.Logger) []string {
		whitelistedDomains := cfg.GetStringSlice("spam.whitelisted_domains")
		if len(whitelistedDomains) > 0 {
			logger.Info("Loaded whitelisted domains", zap.Strings("domains", whitelistedDomains))
		}
		return whitelistedDomains
	}); err != nil {
		return nil, err
	}

	// Register spam threshold
	if err := container.Provide(func(cfg *config.Config) float64 {
		return cfg.GetFloat64("spam.threshold")
	}); err != nil {
		return nil, err
	}

	// Register spam filter service
	if err := container.Provide(core.NewSpamFilterService); err != nil {
		return nil, err
	}

	// Register email filter
	if err := container.Provide(func(f *factory.FilterFactory) (ports.EmailFilter, error) {
		return f.CreateEmailFilter()
	}); err != nil {
		return nil, err
	}

	return container, nil
}
	// Register text processor
	if err := container.Provide(func(f *factory.TextProcessorFactory) *utils.TextProcessor {
		return f.CreateTextProcessor()
	}); err != nil {
		return nil, err
	}
