package factory

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/adapters/bedrock"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// BedrockFactory creates Bedrock LLM clients
type BedrockFactory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewBedrockFactory creates a new Bedrock factory
func NewBedrockFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *BedrockFactory {
	return &BedrockFactory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateLLMClient creates a Bedrock LLM client
func (f *BedrockFactory) CreateLLMClient() (core.LLMClient, error) {
	// Get Bedrock config
	bedrockCfg := f.cfg.GetBedrock()
	
	// Initialize AWS client
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(bedrockCfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Initialize Bedrock client
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	return bedrock.NewBedrockClient(
		bedrockClient,
		bedrockCfg.ModelID,
		bedrockCfg.MaxTokens,
		bedrockCfg.Temperature,
		bedrockCfg.TopP,
		bedrockCfg.MaxBodySize,
		f.logger,
		f.textProcessor,
	), nil
}
