package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/openai"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// OpenAIFactory creates OpenAI LLM clients
type OpenAIFactory struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewOpenAIFactory creates a new OpenAI factory
func NewOpenAIFactory(cfg *config.Config, logger *zap.Logger) *OpenAIFactory {
	return &OpenAIFactory{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateLLMClient creates an OpenAI LLM client
func (f *OpenAIFactory) CreateLLMClient() (core.LLMClient, error) {
	apiKey := f.cfg.GetString("openai.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}
	
	factory := openai.NewFactory(
		apiKey,
		f.cfg.GetString("openai.model_name"),
		f.cfg.GetInt("openai.max_tokens"),
		float32(f.cfg.GetFloat64("openai.temperature")),
		float32(f.cfg.GetFloat64("openai.top_p")),
		f.cfg.GetInt("openai.max_body_size"),
		f.logger,
	)
	return factory.CreateLLMClient()
}
