package core

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SpamFilterService is the core service for spam detection
type SpamFilterService struct {
	llmClient         LLMClient
	cacheRepo         CacheRepository
	logger            *zap.Logger
	cacheEnabled      bool
	cacheTTL          time.Duration
	spamThreshold     float64
	whitelistedDomains []string
}

// NewSpamFilterService creates a new spam filter service
func NewSpamFilterService(
	llmClient LLMClient,
	cacheRepo CacheRepository,
	logger *zap.Logger,
	cacheEnabled bool,
	cacheTTL time.Duration,
	spamThreshold float64,
	whitelistedDomains []string,
) *SpamFilterService {
	return &SpamFilterService{
		llmClient:         llmClient,
		cacheRepo:         cacheRepo,
		logger:            logger,
		cacheEnabled:      cacheEnabled,
		cacheTTL:          cacheTTL,
		spamThreshold:     spamThreshold,
		whitelistedDomains: whitelistedDomains,
	}
}

// AnalyzeEmail analyzes an email to determine if it's spam
func (s *SpamFilterService) AnalyzeEmail(ctx context.Context, email *Email) (*SpamAnalysisResult, error) {
	// Check if sender domain is whitelisted
	if s.isWhitelisted(email.From) {
		s.logger.Info("Email from whitelisted domain, skipping spam check",
			zap.String("from", email.From))
		return &SpamAnalysisResult{
			IsSpam:      false,
			Score:       0.0,
			Confidence:  1.0,
			Explanation: "Sender domain is whitelisted",
			AnalyzedAt:  time.Now(),
			ModelUsed:   "whitelist",
		}, nil
	}

	// Check cache if enabled
	if s.cacheEnabled && s.cacheRepo != nil {
		if result, found := s.cacheRepo.Get(email.From); found {
			s.logger.Info("Using cached result for sender",
				zap.String("from", email.From),
				zap.Bool("is_spam", result.IsSpam),
				zap.Float64("score", result.Score))
			return result, nil
		}
	}

	// Analyze with LLM
	result, err := s.llmClient.AnalyzeEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Apply threshold
	result.IsSpam = result.Score >= s.spamThreshold

	// Cache result if enabled
	if s.cacheEnabled && s.cacheRepo != nil {
		s.cacheRepo.Set(email.From, result, s.cacheTTL)
		s.logger.Debug("Cached result for sender",
			zap.String("from", email.From),
			zap.Duration("ttl", s.cacheTTL))
	}

	return result, nil
}

// isWhitelisted checks if the sender's domain is in the whitelist
func (s *SpamFilterService) isWhitelisted(from string) bool {
	if len(s.whitelistedDomains) == 0 {
		return false
	}

	// Extract domain from email address
	parts := strings.Split(from, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])

	// Check if domain is in whitelist
	for _, whitelisted := range s.whitelistedDomains {
		if strings.ToLower(whitelisted) == domain {
			return true
		}
	}

	return false
}
