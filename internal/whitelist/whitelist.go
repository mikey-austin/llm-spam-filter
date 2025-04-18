package whitelist

import (
	"strings"

	"go.uber.org/zap"
)

// Checker provides functionality to check if email domains are whitelisted
type Checker struct {
	domains []string
	logger  *zap.Logger
}

// NewChecker creates a new whitelist checker
func NewChecker(domains []string, logger *zap.Logger) *Checker {
	// Normalize domains (lowercase)
	normalizedDomains := make([]string, len(domains))
	for i, domain := range domains {
		normalizedDomains[i] = strings.ToLower(strings.TrimSpace(domain))
	}

	if len(normalizedDomains) > 0 && logger != nil {
		logger.Info("Initialized whitelist checker", zap.Strings("domains", normalizedDomains))
	}

	return &Checker{
		domains: normalizedDomains,
		logger:  logger,
	}
}

// IsWhitelisted checks if the sender's domain is in the whitelist
func (c *Checker) IsWhitelisted(from string) bool {
	if len(c.domains) == 0 {
		return false
	}

	// Extract domain from email address
	parts := strings.Split(from, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])

	// Check if domain is in whitelist
	for _, whitelisted := range c.domains {
		if whitelisted == domain {
			if c.logger != nil {
				c.logger.Debug("Domain is whitelisted", 
					zap.String("domain", domain),
					zap.String("email", from))
			}
			return true
		}
	}

	return false
}
