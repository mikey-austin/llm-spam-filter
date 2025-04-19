package factory

import (
	"fmt"

	"github.com/mikey/llm-spam-filter/internal/adapters/gemini"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// GeminiFactory creates Gemini LLM clients
type GeminiFactory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewGeminiFactory creates a new Gemini factory
func NewGeminiFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *GeminiFactory {
	return &GeminiFactory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateLLMClient creates a Gemini LLM client
func (f *GeminiFactory) CreateLLMClient() (core.LLMClient, error) {
	// Get Gemini config
	geminiCfg := f.cfg.GetGemini()
	
	if geminiCfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	
	factory := gemini.NewFactory(f.cfg, f.logger, f.textProcessor)
	return factory.CreateClient()
}
