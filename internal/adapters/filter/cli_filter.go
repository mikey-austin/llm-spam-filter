package filter

import (
	"context"
	"fmt"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
	"time"
)

type CliFilter struct {
	service *core.SpamFilterService
	logger  *zap.Logger
}

func NewCliFilter(service *core.SpamFilterService, logger *zap.Logger) (*CliFilter, error) {
	return &CliFilter{
		service: service,
		logger:  logger,
	}, nil
}

func (f *CliFilter) ProcessEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	f.logger.Debug("Processing email", zap.String("sender", email.From))

	// Print email summary
	fmt.Printf("\n=== Email Summary ===\n")
	fmt.Printf("From: %s\n", email.From)
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Printf("Body length: %d bytes\n", len(email.Body))
	fmt.Printf("\n")

	// Analyze email
	fmt.Printf("=== Analysis ===\n")
	startTime := time.Now()
	result, err := f.service.AnalyzeEmail(ctx, email)
	if err != nil {
		f.logger.Fatal("Failed to analyze email", zap.Error(err))
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

	return nil, nil
}

func (f *CliFilter) Start() error {
	return nil
}

func (f *CliFilter) Stop() error {
	return nil
}
