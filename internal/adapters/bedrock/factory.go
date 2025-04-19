package bedrock

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/utils"
	"go.uber.org/zap"
)

// Factory creates Bedrock clients
type Factory struct {
	cfg          *config.Config
	logger       *zap.Logger
	textProcessor *utils.TextProcessor
}

// NewFactory creates a new Bedrock factory
func NewFactory(cfg *config.Config, logger *zap.Logger, textProcessor *utils.TextProcessor) *Factory {
	return &Factory{
		cfg:          cfg,
		logger:       logger,
		textProcessor: textProcessor,
	}
}

// CreateClient creates a new Bedrock client
func (f *Factory) CreateClient() (*BedrockClient, error) {
	// Get Bedrock config
	bedrockCfg := f.cfg.GetBedrock()
	
	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), 
		awsconfig.WithRegion(bedrockCfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	
	// Create Bedrock client
	client := bedrockruntime.NewFromConfig(awsCfg)
	
	return NewBedrockClient(
		client,
		bedrockCfg.ModelID,
		bedrockCfg.MaxTokens,
		bedrockCfg.Temperature,
		bedrockCfg.TopP,
		bedrockCfg.MaxBodySize,
		f.logger,
		f.textProcessor,
	), nil
}
