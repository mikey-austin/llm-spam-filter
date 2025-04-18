package ports

import (
	"context"

	"github.com/mikey/llm-spam-filter/internal/core"
)

// CacheRepository defines the interface for caching spam analysis results
type CacheRepository interface {
	// Get retrieves a cached entry for a sender
	Get(ctx context.Context, senderEmail string) (*core.CacheEntry, error)
	
	// Set stores a cache entry
	Set(ctx context.Context, entry *core.CacheEntry) error
	
	// Delete removes a cache entry
	Delete(ctx context.Context, senderEmail string) error
	
	// Cleanup removes expired entries
	Cleanup(ctx context.Context) error
}
