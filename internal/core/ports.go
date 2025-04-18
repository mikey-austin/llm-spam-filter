package core

import (
	"context"
)

// LLMClient defines the interface for interacting with LLM services
type LLMClient interface {
	// AnalyzeEmail analyzes an email to determine if it's spam
	AnalyzeEmail(ctx context.Context, email *Email) (*SpamAnalysisResult, error)
}

// CacheRepository defines the interface for caching spam analysis results
type CacheRepository interface {
	// Get retrieves a cached entry for a sender
	Get(ctx context.Context, senderEmail string) (*CacheEntry, error)
	
	// Set stores a cache entry
	Set(ctx context.Context, entry *CacheEntry) error
	
	// Delete removes a cache entry
	Delete(ctx context.Context, senderEmail string) error
	
	// Cleanup removes expired entries
	Cleanup(ctx context.Context) error
}
