package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
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
	textProcessor *utils.TextProcessor
}

// SpamAnalysisResponse represents the structured response from the LLM
type SpamAnalysisResponse struct {
	IsSpam      bool    `json:"is_spam"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// NewBedrockClient creates a new Bedrock client
func NewBedrockClient(
	client *bedrockruntime.Client,
	modelID string,
	maxTokens int,
	temperature float32,
	topP float32,
	maxBodySize int,
	logger *zap.Logger,
	textProcessor *utils.TextProcessor,
) *BedrockClient {
	return &BedrockClient{
		client:       client,
		modelID:      modelID,
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

// isAnthropicModel checks if the model is an Anthropic Claude model

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
	
	// Process the body (truncate and sanitize)
	processedBody := c.textProcessor.ProcessText(email.Body, c.maxBodySize)
	
	prompt := fmt.Sprintf(c.promptFormat, email.From, to, email.Subject, processedBody)
	
	// Create the request based on the model
	var payload []byte
	var err error
	
	if c.isAnthropicModel() {
		// Anthropic Claude models
		payload, err = json.Marshal(map[string]interface{}{
			"prompt":      prompt,
			"max_tokens_to_sample": c.maxTokens,
			"temperature": c.temperature,
			"top_p":       c.topP,
		})
	} else if c.isAmazonTitanModel() {
		// Amazon Titan models
		payload, err = json.Marshal(map[string]interface{}{
			"inputText":  prompt,
			"textGenerationConfig": map[string]interface{}{
				"maxTokenCount": c.maxTokens,
				"temperature":   c.temperature,
				"topP":          c.topP,
			},
		})
	} else {
		// Default to a generic format
		payload, err = json.Marshal(map[string]interface{}{
			"prompt":      prompt,
			"max_tokens":  c.maxTokens,
			"temperature": c.temperature,
			"top_p":       c.topP,
		})
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	
	// Call Bedrock API
	resp, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:   &c.modelID,
		Body:      payload,
		Accept:    aws.String("application/json"),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Bedrock model: %w", err)
	}
	
	// Parse the response based on the model
	var responseText string
	
	if c.isAnthropicModel() {
		// Anthropic Claude models
		var claudeResp struct {
			Completion string `json:"completion"`
		}
		if err := json.Unmarshal(resp.Body, &claudeResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Claude response: %w", err)
		}
		responseText = claudeResp.Completion
	} else if c.isAmazonTitanModel() {
		// Amazon Titan models
		var titanResp struct {
			Results []struct {
				OutputText string `json:"outputText"`
			} `json:"results"`
		}
		if err := json.Unmarshal(resp.Body, &titanResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Titan response: %w", err)
		}
		if len(titanResp.Results) > 0 {
			responseText = titanResp.Results[0].OutputText
		} else {
			return nil, fmt.Errorf("empty response from Titan model")
		}
	} else {
		// Try a generic approach
		var genericResp struct {
			Output string `json:"output"`
			Text   string `json:"text"`
			Response string `json:"response"`
		}
		if err := json.Unmarshal(resp.Body, &genericResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal generic response: %w", err)
		}
		
		// Try different fields
		if genericResp.Output != "" {
			responseText = genericResp.Output
		} else if genericResp.Text != "" {
			responseText = genericResp.Text
		} else if genericResp.Response != "" {
			responseText = genericResp.Response
		} else {
			// Just use the raw response as a string
			responseText = string(resp.Body)
		}
	}

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
		ModelUsed:   c.modelID,
	}
	
	return result, nil
}

// isAnthropicModel checks if the model is an Anthropic Claude model
func (c *BedrockClient) isAnthropicModel() bool {
	return strings.HasPrefix(c.modelID, "anthropic.claude")
}

// isAmazonTitanModel checks if the model is an Amazon Titan model
func (c *BedrockClient) isAmazonTitanModel() bool {
	return strings.HasPrefix(c.modelID, "amazon.titan")
}
