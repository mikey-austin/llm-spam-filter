package core

import (
	"time"
)

// Email represents an email message to be analyzed
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

// CacheEntry represents a cached spam analysis result
type CacheEntry struct {
	SenderEmail string
	IsSpam      bool
	Score       float64
	LastSeen    time.Time
	ExpiresAt   time.Time
}
