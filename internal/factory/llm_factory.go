package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/bedrock"
	"github.com/mikey/llm-spam-filter/internal/adapters/gemini"
	"github.com/mikey/llm-spam-filter/internal/adapters/openai"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// LLMFactory creates LLM clients
type LLMFactory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewLLMFactory creates a new LLM factory
func NewLLMFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *LLMFactory {
	return &LLMFactory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateLLMClient creates a new LLM client based on the configuration
func (f *LLMFactory) CreateLLMClient() (core.LLMClient, error) {
	llmConfig := f.cfg.GetLLM()
	
	switch llmConfig.Provider {
	case "bedrock":
		factory := bedrock.NewFactory(f.cfg, f.logger, f.textProcessor)
		return factory.CreateClient()
	case "gemini":
		factory := gemini.NewFactory(f.cfg, f.logger, f.textProcessor)
		return factory.CreateClient()
	case "openai":
		factory := openai.NewFactory(f.cfg, f.logger, f.textProcessor)
		client, err := factory.CreateLLMClient()
		return client, err
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llmConfig.Provider)
	}
}
