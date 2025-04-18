package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

var (
	// ErrNotFound is returned when a cache entry is not found
	ErrNotFound = errors.New("cache entry not found")
	// ErrExpired is returned when a cache entry has expired
	ErrExpired = errors.New("cache entry expired")
)

// MemoryCache is an in-memory implementation of the CacheRepository interface
type MemoryCache struct {
	entries     map[string]*core.CacheEntry
	mu          sync.RWMutex
	logger      *zap.Logger
	cleanupFreq time.Duration
	stopCh      chan struct{}
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(logger *zap.Logger, cleanupFreq time.Duration) *MemoryCache {
	cache := &MemoryCache{
		entries:     make(map[string]*core.CacheEntry),
		logger:      logger,
		cleanupFreq: cleanupFreq,
		stopCh:      make(chan struct{}),
	}
	
	// Start background cleanup
	go cache.startCleanupTask()
	
	return cache
}

// Get retrieves a cached entry for a sender
func (c *MemoryCache) Get(ctx context.Context, senderEmail string) (*core.CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, ok := c.entries[senderEmail]
	if !ok {
		return nil, ErrNotFound
	}
	
	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, ErrExpired
	}
	
	return entry, nil
}

// Set stores a cache entry
func (c *MemoryCache) Set(ctx context.Context, entry *core.CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries[entry.SenderEmail] = entry
	return nil
}

// Delete removes a cache entry
func (c *MemoryCache) Delete(ctx context.Context, senderEmail string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.entries, senderEmail)
	return nil
}

// Cleanup removes expired entries
func (c *MemoryCache) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	expiredCount := 0
	
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
			expiredCount++
		}
	}
	
	c.logger.Debug("Cleaned up expired cache entries", zap.Int("expired_count", expiredCount))
	return nil
}

// startCleanupTask starts a background task to clean up expired entries
func (c *MemoryCache) startCleanupTask() {
	ticker := time.NewTicker(c.cleanupFreq)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := c.Cleanup(context.Background()); err != nil {
				c.logger.Error("Failed to clean up cache", zap.Error(err))
			}
		case <-c.stopCh:
			return
		}
	}
}

// Stop stops the background cleanup task
func (c *MemoryCache) Stop() {
	close(c.stopCh)
}
