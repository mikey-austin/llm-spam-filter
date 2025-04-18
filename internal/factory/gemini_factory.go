package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/gemini"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// GeminiFactory creates Gemini LLM clients
type GeminiFactory struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewGeminiFactory creates a new Gemini factory
func NewGeminiFactory(cfg *config.Config, logger *zap.Logger) *GeminiFactory {
	return &GeminiFactory{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateLLMClient creates a Gemini LLM client
func (f *GeminiFactory) CreateLLMClient() (core.LLMClient, error) {
	apiKey := f.cfg.GetString("gemini.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	
	factory := gemini.NewFactory(
		apiKey,
		f.cfg.GetString("gemini.model_name"),
		f.cfg.GetInt("gemini.max_tokens"),
		float32(f.cfg.GetFloat64("gemini.temperature")),
		float32(f.cfg.GetFloat64("gemini.top_p")),
		f.cfg.GetInt("gemini.max_body_size"),
		f.logger,
	)
	return factory.CreateLLMClient()
}
