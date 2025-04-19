package factory

import (
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// TextProcessorFactory creates text processors
type TextProcessorFactory struct {
	logger *zap.Logger
}

// NewTextProcessorFactory creates a new TextProcessorFactory
func NewTextProcessorFactory(logger *zap.Logger) *TextProcessorFactory {
	return &TextProcessorFactory{
		logger: logger,
	}
}

// CreateTextProcessor creates a new TextProcessor
func (f *TextProcessorFactory) CreateTextProcessor() *utils.TextProcessor {
	return utils.NewTextProcessor(f.logger)
}
