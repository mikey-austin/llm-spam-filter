package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// BedrockClient is an implementation of the LLMClient interface using Amazon Bedrock
type BedrockClient struct {
	client       *bedrockruntime.Client
	modelID      string
	maxTokens    int
	temperature  float32
	topP         float32
	maxBodySize  int
	logger       *zap.Logger
	promptFormat string
}

// BedrockRequest represents the request structure for Bedrock API
type BedrockRequest struct {
	Prompt            string  `json:"prompt"`
	MaxTokens         int     `json:"max_tokens"`
	Temperature       float32 `json:"temperature"`
	TopP              float32 `json:"top_p"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
}

// BedrockResponse represents the response structure from Bedrock API
type BedrockResponse struct {
	Completion string  `json:"completion"`
	StopReason string  `json:"stop_reason"`
}

// SpamAnalysisResponse represents the structured response from the LLM
type SpamAnalysisResponse struct {
	IsSpam      bool    `json:"is_spam"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// NewBedrockClient creates a new Amazon Bedrock client
func NewBedrockClient(
	client *bedrockruntime.Client,
	modelID string,
	maxTokens int,
	temperature float32,
	topP float32,
	maxBodySize int,
	logger *zap.Logger,
) *BedrockClient {
	return &BedrockClient{
		client:      client,
		modelID:     modelID,
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
	}
}

// truncateBody truncates the email body if it exceeds the maximum size
func (c *BedrockClient) truncateBody(body string) string {
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
func (c *BedrockClient) AnalyzeEmail(ctx context.Context, email *core.Email) (*core.SpamAnalysisResult, error) {
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
	
	// Create the request payload
	requestBody := BedrockRequest{
		Prompt:      prompt,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		TopP:        c.topP,
	}
	
	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Call Bedrock API
	response, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        requestBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Bedrock model: %w", err)
	}
	
	// Parse the response
	var bedrockResponse BedrockResponse
	if err := json.Unmarshal(response.Body, &bedrockResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Bedrock response: %w", err)
	}
	
	// Parse the LLM's JSON response
	var analysisResponse SpamAnalysisResponse
	if err := json.Unmarshal([]byte(bedrockResponse.Completion), &analysisResponse); err != nil {
		// Try to extract JSON from the text response
		jsonStart := 0
		jsonEnd := len(bedrockResponse.Completion)
		
		// Find JSON start
		for i := 0; i < len(bedrockResponse.Completion); i++ {
			if bedrockResponse.Completion[i] == '{' {
				jsonStart = i
				break
			}
		}
		
		// Find JSON end
		for i := len(bedrockResponse.Completion) - 1; i >= 0; i-- {
			if bedrockResponse.Completion[i] == '}' {
				jsonEnd = i + 1
				break
			}
		}
		
		if jsonStart < jsonEnd {
			jsonStr := bedrockResponse.Completion[jsonStart:jsonEnd]
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
		ModelUsed:   c.modelID,
		ProcessingID: string(response.ResponseMetadata.RequestID),
	}
	
	return result, nil
}
