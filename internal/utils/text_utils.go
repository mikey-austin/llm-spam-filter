package utils

import (
	"unicode/utf8"

	"go.uber.org/zap"
)

// TextProcessor provides utilities for processing text
type TextProcessor struct {
	logger *zap.Logger
}

// NewTextProcessor creates a new TextProcessor
func NewTextProcessor(logger *zap.Logger) *TextProcessor {
	return &TextProcessor{
		logger: logger,
	}
}

// TruncateText safely truncates text to the specified maximum size
// and ensures the result is valid UTF-8
func (tp *TextProcessor) TruncateText(text string, maxSize int) string {
	// If no limit or text is already within limits, return as is
	if maxSize <= 0 || len(text) <= maxSize {
		return text
	}

	// First truncate to the byte limit
	truncated := text[:maxSize]

	// Ensure the truncated text ends with a valid UTF-8 sequence
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		// Remove bytes until we have valid UTF-8
		truncated = truncated[:len(truncated)-1]
	}

	tp.logger.Debug("Text truncated",
		zap.Int("original_size", len(text)),
		zap.Int("truncated_size", len(truncated)),
		zap.Int("max_size", maxSize))

	return truncated + "\n[... Content truncated due to size limits ...]"
}

// SanitizeUTF8 ensures the string contains only valid UTF-8 characters
func (tp *TextProcessor) SanitizeUTF8(text string) string {
	if utf8.ValidString(text) {
		return text
	}

	// Replace invalid UTF-8 sequences with the Unicode replacement character
	result := make([]rune, 0, len(text))
	for i, r := range text {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(text[i:])
			if size == 1 {
				// Skip invalid UTF-8 sequences
				continue
			}
		}
		result = append(result, r)
	}

	tp.logger.Debug("Text sanitized",
		zap.Int("original_size", len(text)),
		zap.Int("sanitized_size", len(string(result))))

	return string(result)
}

// ProcessText truncates and sanitizes text in one operation
func (tp *TextProcessor) ProcessText(text string, maxSize int) string {
	// First truncate
	truncated := tp.TruncateText(text, maxSize)
	
	// Then sanitize
	sanitized := tp.SanitizeUTF8(truncated)
	
	return sanitized
}
