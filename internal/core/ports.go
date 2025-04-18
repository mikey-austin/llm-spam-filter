package core

import (
	"context"
	"time"
)

// LLMClient defines the interface for interacting with LLM providers
type LLMClient interface {
	// AnalyzeEmail analyzes an email to determine if it's spam
	AnalyzeEmail(ctx context.Context, email *Email) (*SpamAnalysisResult, error)
}

// CacheRepository defines the interface for caching spam analysis results
type CacheRepository interface {
	Get(key string) (*SpamAnalysisResult, bool)
	Set(key string, result *SpamAnalysisResult, ttl time.Duration)
}
