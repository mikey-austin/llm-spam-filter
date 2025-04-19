package di

import (
	"flag"
	"time"

	"go.uber.org/dig"
	"go.uber.org/zap"

	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/factory"
	"github.com/mikey/llm-spam-filter/internal/logging"
	"github.com/mikey/llm-spam-filter/internal/ports"
)

// CLIFlags contains all command line flags for the CLI application
type CLIFlags struct {
	// LLM provider flags
	Provider    string
	MaxTokens   int
	Temperature float64
	TopP        float64
	MaxBodySize int

	// Bedrock flags
	BedrockRegion  string
	BedrockModelID string

	// Gemini flags
	GeminiAPIKey    string
	GeminiModelName string

	// OpenAI flags
	OpenAIAPIKey    string
	OpenAIModelName string

	// Spam detection flags
	SpamThreshold float64

	// Input flags
	InputFile  string
	Verbose    bool
	JSONLog    bool
	ConfigFile string
}

// ParseFlags parses command line flags and returns a CLIFlags struct
func ParseFlags() *CLIFlags {
	flags := &CLIFlags{}

	// LLM provider flags
	flag.StringVar(&flags.Provider, "provider", "bedrock", "LLM provider (bedrock, gemini, openai)")
	flag.IntVar(&flags.MaxTokens, "max-tokens", 1000, "Maximum tokens for LLM response")
	flag.Float64Var(&flags.Temperature, "temperature", 0.1, "Temperature for LLM generation")
	flag.Float64Var(&flags.TopP, "top-p", 0.9, "Top-p for LLM generation")
	flag.IntVar(&flags.MaxBodySize, "max-body-size", 4096, "Maximum email body size to send to LLM")

	// Bedrock flags
	flag.StringVar(&flags.BedrockRegion, "bedrock-region", "us-east-1", "AWS region for Bedrock")
	flag.StringVar(&flags.BedrockModelID, "bedrock-model", "anthropic.claude-v2", "Bedrock model ID")

	// Gemini flags
	flag.StringVar(&flags.GeminiAPIKey, "gemini-api-key", "", "API key for Google Gemini")
	flag.StringVar(&flags.GeminiModelName, "gemini-model", "gemini-pro", "Gemini model name")

	// OpenAI flags
	flag.StringVar(&flags.OpenAIAPIKey, "openai-api-key", "", "API key for OpenAI")
	flag.StringVar(&flags.OpenAIModelName, "openai-model", "gpt-4", "OpenAI model name")

	// Spam detection flags
	flag.Float64Var(&flags.SpamThreshold, "threshold", 0.7, "Threshold for spam detection")

	// Input flags
	flag.StringVar(&flags.InputFile, "file", "", "Input email file (use stdin if not specified)")
	flag.BoolVar(&flags.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&flags.JSONLog, "json-log", false, "Output logs in JSON format")
	flag.StringVar(&flags.ConfigFile, "config", "", "Path to config file (overrides command line flags)")

	flag.Parse()
	return flags
}

// BuildCLIContainer creates and configures a dependency injection container for the CLI application
func BuildCLIContainer(flags *CLIFlags) (*dig.Container, error) {
	container := dig.New()

	// Register flags
	if err := container.Provide(func() *CLIFlags { return flags }); err != nil {
		return nil, err
	}

	// Register logger
	if err := container.Provide(func(flags *CLIFlags) (*zap.Logger, error) {
		return logging.InitConsoleLogger(flags.Verbose, flags.JSONLog)
	}); err != nil {
		return nil, err
	}

	// Register configuration
	if err := container.Provide(func(flags *CLIFlags, logger *zap.Logger) (*config.Config, error) {
		if flags.ConfigFile != "" {
			cfg, err := config.New()
			if err != nil {
				return nil, err
			}
			logger.Info("Loaded configuration from file", zap.String("file", cfg.GetViper().ConfigFileUsed()))
			return cfg, nil
		}
		
		// Create config from command line flags
		return createConfigFromFlags(flags), nil
	}); err != nil {
		return nil, err
	}

	// Register factories
	if err := container.Provide(factory.NewLLMFactory); err != nil {
		return nil, err
	}
	if err := container.Provide(factory.NewFilterFactory); err != nil {
		return nil, err
	}

	// Register LLM client
	if err := container.Provide(func(f *factory.LLMFactory) (core.LLMClient, error) {
		return f.CreateLLMClient()
	}); err != nil {
		return nil, err
	}

	// Register spam threshold
	if err := container.Provide(func(cfg *config.Config) float64 {
		return cfg.GetFloat64("spam.threshold")
	}); err != nil {
		return nil, err
	}

	// Register empty whitelisted domains for CLI
	if err := container.Provide(func() []string {
		return []string{}
	}); err != nil {
		return nil, err
	}

	// Register spam filter service with no cache
	if err := container.Provide(func(
		llmClient core.LLMClient,
		logger *zap.Logger,
		spamThreshold float64,
		whitelistedDomains []string,
	) *core.SpamFilterService {
		return core.NewSpamFilterService(
			llmClient,
			nil, // No cache for CLI
			logger,
			false, // Cache disabled
			time.Duration(0), // No TTL
			spamThreshold,
			whitelistedDomains,
		)
	}); err != nil {
		return nil, err
	}

	// Register email filter
	if err := container.Provide(func(f *factory.FilterFactory) (ports.EmailFilter, error) {
		return f.CreateEmailFilter()
	}); err != nil {
		return nil, err
	}

	return container, nil
}

// createConfigFromFlags creates a configuration from command line flags
func createConfigFromFlags(flags *CLIFlags) *config.Config {
	v := config.NewEmptyViper()

	// Set some cli specific settings
	v.Set("server.filter_type", "cli")
	v.Set("cli.verbose", flags.Verbose)

	// Set LLM provider
	v.Set("llm.provider", flags.Provider)

	// Set provider-specific configuration
	switch flags.Provider {
	case "bedrock":
		v.Set("bedrock.region", flags.BedrockRegion)
		v.Set("bedrock.model_id", flags.BedrockModelID)
		v.Set("bedrock.max_tokens", flags.MaxTokens)
		v.Set("bedrock.temperature", flags.Temperature)
		v.Set("bedrock.top_p", flags.TopP)
		v.Set("bedrock.max_body_size", flags.MaxBodySize)
	case "gemini":
		v.Set("gemini.api_key", flags.GeminiAPIKey)
		v.Set("gemini.model_name", flags.GeminiModelName)
		v.Set("gemini.max_tokens", flags.MaxTokens)
		v.Set("gemini.temperature", flags.Temperature)
		v.Set("gemini.top_p", flags.TopP)
		v.Set("gemini.max_body_size", flags.MaxBodySize)
	case "openai":
		v.Set("openai.api_key", flags.OpenAIAPIKey)
		v.Set("openai.model_name", flags.OpenAIModelName)
		v.Set("openai.max_tokens", flags.MaxTokens)
		v.Set("openai.temperature", flags.Temperature)
		v.Set("openai.top_p", flags.TopP)
		v.Set("openai.max_body_size", flags.MaxBodySize)
	}

	// Set spam threshold
	v.Set("spam.threshold", flags.SpamThreshold)

	return config.NewFromViper(v)
}
