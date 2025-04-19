package openai

import (
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// Factory creates new instances of OpenAIClient
type Factory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewFactory creates a new factory for OpenAIClient instances
func NewFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *Factory {
	return &Factory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateLLMClient creates a new OpenAIClient
func (f *Factory) CreateLLMClient() (core.LLMClient, error) {
	// Get OpenAI config
	openaiCfg := f.cfg.GetOpenAI()
	
	// Create OpenAI client
	client := openai.NewClient(openaiCfg.APIKey)
	
	return NewOpenAIClient(
		client,
		openaiCfg.ModelName,
		openaiCfg.MaxTokens,
		openaiCfg.Temperature,
		openaiCfg.TopP,
		openaiCfg.MaxBodySize,
		f.logger,
		f.textProcessor,
	), nil
}
