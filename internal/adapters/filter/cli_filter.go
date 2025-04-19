package filter

import (
	"context"
	"fmt"
	"time"

	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// CliFilter implements a command-line interface for spam detection
type CliFilter struct {
	service *core.SpamFilterService
	logger  *zap.Logger
	verbose bool
}

// NewCliFilter creates a new CLI filter
func NewCliFilter(service *core.SpamFilterService, logger *zap.Logger, verbose bool) (*CliFilter, error) {
	return &CliFilter{
		service: service,
		logger:  logger,
		verbose: verbose,
	}, nil
}

// ProcessEmail processes an email and displays the results
func (f *CliFilter) ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	f.logger.Debug("Processing email", zap.String("sender", email.From))

	// Print email summary
	fmt.Printf("\n=== Email Summary ===\n")
	fmt.Printf("From: %s\n", email.From)
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Printf("Body length: %d bytes\n", len(email.Body))
	
	// Print body preview if verbose
	if f.verbose {
		preview := email.Body
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("\nBody preview:\n%s\n", preview)
	}
	
	fmt.Printf("\n")

	// Analyze email
	fmt.Printf("=== Analysis ===\n")
	fmt.Printf("Analyzing email with LLM...\n")
	startTime := time.Now()
	result, err := f.service.AnalyzeEmail(ctx, email)
	if err != nil {
		f.logger.Error("Failed to analyze email", zap.Error(err))
		fmt.Printf("Error: %v\n", err)
		return nil, err
	}
	duration := time.Since(startTime)

	// Print results
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Is spam: %t\n", result.IsSpam)
	fmt.Printf("Spam score: %.4f\n", result.Score)
	fmt.Printf("Confidence: %.4f\n", result.Confidence)
	fmt.Printf("Explanation: %s\n", result.Explanation)
	fmt.Printf("Model used: %s\n", result.ModelUsed)
	fmt.Printf("Processing time: %v\n", duration)

	return result, nil
}

// Start is a no-op for the CLI filter
func (f *CliFilter) Start() error {
	return nil
}

// Stop is a no-op for the CLI filter
func (f *CliFilter) Stop() error {
	return nil
}
