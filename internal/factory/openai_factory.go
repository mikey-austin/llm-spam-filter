package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/openai"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// OpenAIFactory creates OpenAI LLM clients
type OpenAIFactory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewOpenAIFactory creates a new OpenAI factory
func NewOpenAIFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *OpenAIFactory {
	return &OpenAIFactory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateLLMClient creates an OpenAI LLM client
func (f *OpenAIFactory) CreateLLMClient() (core.LLMClient, error) {
	// Get OpenAI config
	openaiCfg := f.cfg.GetOpenAI()
	
	if openaiCfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}
	
	factory := openai.NewFactory(f.cfg, f.logger, f.textProcessor)
	client, err := factory.CreateLLMClient()
	return client, err
}
