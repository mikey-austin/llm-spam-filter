package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// GeminiClient is an implementation of the LLMClient interface using Google Gemini
type GeminiClient struct {
	client       *genai.Client
	model        *genai.GenerativeModel
	modelName    string
	maxTokens    int
	temperature  float32
	topP         float32
	maxBodySize  int
	logger       *zap.Logger
	promptFormat string
}

// SpamAnalysisResponse represents the structured response from the LLM
type SpamAnalysisResponse struct {
	IsSpam      bool    `json:"is_spam"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(
	apiKey string,
	modelName string,
	maxTokens int,
	temperature float32,
	topP float32,
	maxBodySize int,
	logger *zap.Logger,
) (*GeminiClient, error) {
	// Create a new Gemini client
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create a generative model
	model := client.GenerativeModel(modelName)
	model.SetTemperature(float32(temperature))
	model.SetTopP(float32(topP))
	model.SetMaxOutputTokens(int32(maxTokens))

	return &GeminiClient{
		client:      client,
		model:       model,
		modelName:   modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		topP:        topP,
		maxBodySize: maxBodySize,
		logger:      logger,
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
	}, nil
}

// Close closes the Gemini client
func (c *GeminiClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// truncateBody truncates the email body if it exceeds the maximum size
func (c *GeminiClient) truncateBody(body string) string {
	if c.maxBodySize <= 0 || len(body) <= c.maxBodySize {
		return body
	}
	
	truncated := body[:c.maxBodySize]
	c.logger.Debug("Email body truncated",
		zap.Int("original_size", len(body)),
		zap.Int("truncated_size", len(truncated)),
		zap.Int("max_size", c.maxBodySize))
	
	return truncated + "\n[... Content truncated due to size limits ...]"
}

// AnalyzeEmail analyzes an email to determine if it's spam
func (c *GeminiClient) AnalyzeEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
	// Format the prompt with email details
	to := ""
	if len(email.To) > 0 {
		to = email.To[0]
		if len(email.To) > 1 {
			to += fmt.Sprintf(" and %d others", len(email.To)-1)
		}
	}
	
	// Truncate the body if needed
	truncatedBody := c.truncateBody(email.Body)
	
	prompt := fmt.Sprintf(c.promptFormat, email.From, to, email.Subject, truncatedBody)
	
	// Call Gemini API
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content with Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Extract the response text
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

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
	}
	
	return result, nil
}
