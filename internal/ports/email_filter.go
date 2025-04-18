package ports

import (
	"context"

	"github.com/mikey/llm-spam-filter/internal/core"
)

// EmailFilter defines the interface for email filtering
type EmailFilter interface {
	// ProcessEmail processes an email and returns the filtering result
	ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error)
	
	// Start starts the email filter service
	Start() error
	
	// Stop stops the email filter service
	Stop() error
}
