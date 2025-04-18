package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mikey/llm-spam-filter/internal/adapters/cache"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// CacheFactory creates cache repositories based on configuration
type CacheFactory struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewCacheFactory creates a new cache factory
func NewCacheFactory(cfg *config.Config, logger *zap.Logger) *CacheFactory {
	return &CacheFactory{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateCacheRepository creates a cache repository based on the configuration
func (f *CacheFactory) CreateCacheRepository() (core.CacheRepository, error) {
	cacheType := f.cfg.GetString("cache.type")
	cleanupFreq, err := f.cfg.GetDuration("cache.cleanup_frequency")
	if err != nil {
		return nil, fmt.Errorf("invalid cache cleanup frequency: %w", err)
	}

	switch cacheType {
	case "memory":
		return cache.NewMemoryCache(f.logger, cleanupFreq), nil
	case "sqlite":
		sqlitePath := f.cfg.GetString("cache.sqlite_path")
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create SQLite directory: %w", err)
		}
		return cache.NewSQLiteCache(sqlitePath, f.logger, cleanupFreq)
	case "mysql":
		mysqlDSN := f.cfg.GetString("cache.mysql_dsn")
		return cache.NewMySQLCache(mysqlDSN, f.logger, cleanupFreq)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
}

// GetCacheTTL returns the configured cache TTL
func (f *CacheFactory) GetCacheTTL() (time.Duration, error) {
	return f.cfg.GetDuration("cache.ttl")
}

// IsCacheEnabled returns whether caching is enabled
func (f *CacheFactory) IsCacheEnabled() bool {
	return f.cfg.GetBool("cache.enabled")
}
