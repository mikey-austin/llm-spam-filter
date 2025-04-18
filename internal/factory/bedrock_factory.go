package factory

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/adapters/bedrock"
	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
)

// BedrockFactory creates Bedrock LLM clients
type BedrockFactory struct {
	cfg    *config.Config
	logger *zap.Logger
}

// NewBedrockFactory creates a new Bedrock factory
func NewBedrockFactory(cfg *config.Config, logger *zap.Logger) *BedrockFactory {
	return &BedrockFactory{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateLLMClient creates a Bedrock LLM client
func (f *BedrockFactory) CreateLLMClient() (core.LLMClient, error) {
	// Initialize AWS client
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(f.cfg.GetString("bedrock.region")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Initialize Bedrock client
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	return bedrock.NewBedrockClient(
		bedrockClient,
		f.cfg.GetString("bedrock.model_id"),
		f.cfg.GetInt("bedrock.max_tokens"),
		float32(f.cfg.GetFloat64("bedrock.temperature")),
		float32(f.cfg.GetFloat64("bedrock.top_p")),
		f.cfg.GetInt("bedrock.max_body_size"),
		f.logger,
	), nil
}
