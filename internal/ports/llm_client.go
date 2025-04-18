package ports

import (
	"context"

	"github.com/mikey/llm-spam-filter/internal/core"
)

// LLMClient defines the interface for interacting with LLM services
type LLMClient interface {
	// AnalyzeEmail analyzes an email to determine if it's spam
	AnalyzeEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error)
}
