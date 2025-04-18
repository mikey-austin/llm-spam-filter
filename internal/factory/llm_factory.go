package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// LLMFactory creates LLM clients based on configuration
type LLMFactory struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewLLMFactory creates a new LLM factory
func NewLLMFactory(cfg *config.Config, logger *zap.Logger) *LLMFactory {
	return &LLMFactory{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateLLMClient creates an LLM client based on the configuration
func (f *LLMFactory) CreateLLMClient() (core.LLMClient, error) {
	provider := f.cfg.GetString("llm.provider")
	
	switch provider {
	case "bedrock":
		factory := NewBedrockFactory(f.cfg, f.logger)
		return factory.CreateLLMClient()
	case "gemini":
		factory := NewGeminiFactory(f.cfg, f.logger)
		return factory.CreateLLMClient()
	case "openai":
		factory := NewOpenAIFactory(f.cfg, f.logger)
		return factory.CreateLLMClient()
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}
