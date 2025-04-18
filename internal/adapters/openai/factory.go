package openai

import (
	"github.com/mikey/llm-spam-filter/internal/ports"
	"go.uber.org/zap"
)

// Factory creates new instances of OpenAIClient
type Factory struct {
	apiKey      string
	modelName   string
	maxTokens   int
	temperature float32
	topP        float32
	maxBodySize int
	logger      *zap.Logger
}

// NewFactory creates a new factory for OpenAIClient instances
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

// CreateLLMClient creates a new OpenAIClient
func (f *Factory) CreateLLMClient() (ports.LLMClient, error) {
	return NewOpenAIClient(
		f.apiKey,
		f.modelName,
		f.maxTokens,
		f.temperature,
		f.topP,
		f.maxBodySize,
		f.logger,
	), nil
}
