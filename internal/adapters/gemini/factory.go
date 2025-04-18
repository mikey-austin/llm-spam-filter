package gemini

import (
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// Factory creates new instances of GeminiClient
type Factory struct {
	apiKey      string
	modelName   string
	maxTokens   int
	temperature float32
	topP        float32
	maxBodySize int
	logger      *zap.Logger
}

// NewFactory creates a new factory for GeminiClient instances
func NewFactory(
	apiKey string,
	modelName string,
	maxTokens int,
	temperature float32,
	topP float32,
	maxBodySize int,
	logger *zap.Logger,
) *Factory {
	return &Factory{
		apiKey:      apiKey,
		modelName:   modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		topP:        topP,
		maxBodySize: maxBodySize,
		logger:      logger,
	}
}

// CreateLLMClient creates a new GeminiClient
func (f *Factory) CreateLLMClient() (core.LLMClient, error) {
	return NewGeminiClient(
		f.apiKey,
		f.modelName,
		f.maxTokens,
		f.temperature,
		f.topP,
		f.maxBodySize,
		f.logger,
	)
}
