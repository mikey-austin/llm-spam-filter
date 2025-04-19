package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// Factory creates Gemini clients
type Factory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewFactory creates a new Gemini factory
func NewFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *Factory {
	return &Factory{
		cfg:    cfg,
		logger: logger,
		textProcessor: textProcessor,
	}
}

// CreateClient creates a new Gemini client
func (f *Factory) CreateClient() (*GeminiClient, error) {
	// Create Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(f.cfg.Gemini.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	
	return NewGeminiClient(
		client,
		f.cfg.Gemini.ModelName,
		f.cfg.Gemini.MaxTokens,
		f.cfg.Gemini.Temperature,
		f.cfg.Gemini.TopP,
		f.cfg.Gemini.MaxBodySize,
		f.logger,
		f.textProcessor,
	)
}
