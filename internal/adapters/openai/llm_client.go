package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// OpenAIClient is an implementation of the LLMClient interface using OpenAI
type OpenAIClient struct {
	client       *openai.Client
	modelName    string
	maxTokens    int
	temperature  float32
	topP         float32
	maxBodySize  int
	logger       *zap.Logger
	promptFormat string
	textProcessor *utils.TextProcessor
}

// SpamAnalysisResponse represents the structured response from the LLM
type SpamAnalysisResponse struct {
	IsSpam      bool    `json:"is_spam"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(
	client *openai.Client,
	modelName string,
	maxTokens int,
	temperature float32,
	topP float32,
	maxBodySize int,
	logger *zap.Logger,
	textProcessor *utils.TextProcessor,
) *OpenAIClient {
	return &OpenAIClient{
		client:       client,
		modelName:    modelName,
		maxTokens:    maxTokens,
		temperature:  temperature,
		topP:         topP,
		maxBodySize:  maxBodySize,
		logger:       logger,
		textProcessor: textProcessor,
		promptFormat: `You are a spam detection system. Analyze the following email and determine if it's spam.
Respond with a JSON object containing:
- is_spam: boolean (true if spam, false if not)
- score: number between 0 and 1 (higher means more likely to be spam)
- confidence: number between 0 and 1 (how confident you are in your assessment)
- explanation: string (brief explanation of why you think it's spam or not)

Email:
From: %s
To: %s
Subject: %s
Body:
%s

Respond only with the JSON object and nothing else.`,
	}
}

// AnalyzeEmail analyzes an email to determine if it's spam
func (c *OpenAIClient) AnalyzeEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	// Format the prompt with email details
	to := ""
	if len(email.To) > 0 {
		to = email.To[0]
		if len(email.To) > 1 {
			to += fmt.Sprintf(" and %d others", len(email.To)-1)
		}
	}
	
	// Process the body (truncate and sanitize)
	processedBody := c.textProcessor.ProcessText(email.Body, c.maxBodySize)
	
	prompt := fmt.Sprintf(c.promptFormat, email.From, to, email.Subject, processedBody)
	
	// Create the request
	req := openai.ChatCompletionRequest{
		Model:       c.modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a spam detection system. Respond only with JSON.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   c.maxTokens,
		Temperature: float32(c.temperature),
		TopP:        float32(c.topP),
	}
	
	// Add response format if supported by the client version
	responseFormat := openai.ChatCompletionResponseFormat{
		Type: "json",
	}
	req.ResponseFormat = &responseFormat
	
	// Call OpenAI API
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion with OpenAI: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI")
	}

	// Extract the response text
	responseText := resp.Choices[0].Message.Content

	// Parse the LLM's JSON response
	var analysisResponse SpamAnalysisResponse
	if err := json.Unmarshal([]byte(responseText), &analysisResponse); err != nil {
		// Try to extract JSON from the text response
		jsonStart := 0
		jsonEnd := len(responseText)
		
		// Find JSON start
		for i := 0; i < len(responseText); i++ {
			if responseText[i] == '{' {
				jsonStart = i
				break
			}
		}
		
		// Find JSON end
		for i := len(responseText) - 1; i >= 0; i-- {
			if responseText[i] == '}' {
				jsonEnd = i + 1
				break
			}
		}
		
		if jsonStart < jsonEnd {
			jsonStr := responseText[jsonStart:jsonEnd]
			if err := json.Unmarshal([]byte(jsonStr), &analysisResponse); err != nil {
				return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to extract JSON from LLM response: %w", err)
		}
	}
	
	// Create the result
	result := &core.SpamAnalysisResult{
		IsSpam:      analysisResponse.IsSpam,
		Score:       analysisResponse.Score,
		Confidence:  analysisResponse.Confidence,
		Explanation: analysisResponse.Explanation,
		AnalyzedAt:  time.Now(),
		ModelUsed:   c.modelName,
		ProcessingID: resp.ID,
	}
	
	return result, nil
}
