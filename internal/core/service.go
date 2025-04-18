package core

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SpamFilterService is the core service for spam detection
type SpamFilterService struct {
	llmClient          LLMClient
	cache              CacheRepository
	logger             *zap.Logger
	cacheEnabled       bool
	cacheTTL           time.Duration
	threshold          float64
	whitelistedDomains []string
}

// NewSpamFilterService creates a new spam filter service
func NewSpamFilterService(
	llmClient LLMClient,
	cache CacheRepository,
	logger *zap.Logger,
	cacheEnabled bool,
	cacheTTL time.Duration,
	threshold float64,
	whitelistedDomains []string,
) *SpamFilterService {
	return &SpamFilterService{
		llmClient:          llmClient,
		cache:              cache,
		logger:             logger,
		cacheEnabled:       cacheEnabled,
		cacheTTL:           cacheTTL,
		threshold:          threshold,
		whitelistedDomains: whitelistedDomains,
	}
}

// isDomainWhitelisted checks if the sender's domain is in the whitelist
func (s *SpamFilterService) isDomainWhitelisted(email string) bool {
	// Extract domain from email address
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	domain := strings.ToLower(parts[1])
	
	// Check if domain is in whitelist
	for _, whitelistedDomain := range s.whitelistedDomains {
		if strings.EqualFold(domain, whitelistedDomain) {
			return true
		}
	}
	
	return false
}

// AnalyzeEmail checks if an email is spam
func (s *SpamFilterService) AnalyzeEmail(ctx context.Context, email *Email) (*SpamAnalysisResult, error) {
	// Check whitelist first
	if s.isDomainWhitelisted(email.From) {
		s.logger.Info("Skipping spam check for whitelisted domain", 
			zap.String("sender", email.From),
			zap.String("action", "whitelist_bypass"))
		
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
	if s.cacheEnabled {
		if entry, err := s.cache.Get(ctx, email.From); err == nil {
			s.logger.Debug("Cache hit for sender", zap.String("sender", email.From))
			return &SpamAnalysisResult{
				IsSpam:      entry.IsSpam,
				Score:       entry.Score,
				Confidence:  1.0, // High confidence since it's cached
				Explanation: "Result from cache",
				AnalyzedAt:  time.Now(),
				ModelUsed:   "cache",
			}, nil
		}
	}

	// Call LLM for analysis
	result, err := s.llmClient.AnalyzeEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Update cache with result if enabled
	if s.cacheEnabled {
		expiresAt := time.Now().Add(s.cacheTTL)
		entry := &CacheEntry{
			SenderEmail: email.From,
			IsSpam:      result.IsSpam,
			Score:       result.Score,
			LastSeen:    time.Now(),
			ExpiresAt:   expiresAt,
		}
		if err := s.cache.Set(ctx, entry); err != nil {
			s.logger.Error("Failed to update cache", zap.Error(err))
		}
	}

	return result, nil
}

// IsSpam determines if an email is spam based on the threshold
func (s *SpamFilterService) IsSpam(result *SpamAnalysisResult) bool {
	return result.Score >= s.threshold
}
