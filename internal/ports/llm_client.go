package ports

import (
	"context"
	"time"
)

// Email represents an email message
type Email struct {
	From    string
	To      []string
	Subject string
	Body    string
	Headers map[string][]string
}

// SpamAnalysisResult represents the result of spam analysis
type SpamAnalysisResult struct {
	IsSpam       bool
	Score        float64
	Confidence   float64
	Explanation  string
	AnalyzedAt   time.Time
	ModelUsed    string
	ProcessingID string
}

// LLMClient defines the interface for interacting with LLM providers
type LLMClient interface {
	// AnalyzeEmail analyzes an email to determine if it's spam
	AnalyzeEmail(ctx context.Context, email *Email) (*SpamAnalysisResult, error)
}
