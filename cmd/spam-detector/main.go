package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/mikey/llm-spam-filter/internal/config"
	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/factory"
	"github.com/mikey/llm-spam-filter/internal/logging"
	"github.com/mikey/llm-spam-filter/internal/whitelist"
	"go.uber.org/zap"
)

var (
	// LLM provider flags
	provider    = flag.String("provider", "bedrock", "LLM provider (bedrock, gemini, openai)")
	maxTokens   = flag.Int("max-tokens", 1000, "Maximum tokens for LLM response")
	temperature = flag.Float64("temperature", 0.1, "Temperature for LLM generation")
	topP        = flag.Float64("top-p", 0.9, "Top-p for LLM generation")
	maxBodySize = flag.Int("max-body-size", 4096, "Maximum email body size to send to LLM")

	// Bedrock flags
	bedrockRegion  = flag.String("bedrock-region", "us-east-1", "AWS region for Bedrock")
	bedrockModelID = flag.String("bedrock-model", "anthropic.claude-v2", "Bedrock model ID")

	// Gemini flags
	geminiAPIKey   = flag.String("gemini-api-key", "", "API key for Google Gemini")
	geminiModelName = flag.String("gemini-model", "gemini-pro", "Gemini model name")

	// OpenAI flags
	openaiAPIKey   = flag.String("openai-api-key", "", "API key for OpenAI")
	openaiModelName = flag.String("openai-model", "gpt-4", "OpenAI model name")

	// Spam detection flags
	spamThreshold = flag.Float64("threshold", 0.7, "Threshold for spam detection")
	whitelistDomains = flag.String("whitelist", "", "Comma-separated list of whitelisted domains")

	// Input flags
	inputFile = flag.String("file", "", "Input email file (use stdin if not specified)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
	jsonLog   = flag.Bool("json-log", false, "Output logs in JSON format")
	configFile = flag.String("config", "", "Path to config file (overrides command line flags)")
)

func main() {
	flag.Parse()

	var cfg *config.Config
	var err error

	// Initialize logger
	logger, err := logging.InitConsoleLogger(*verbose, *jsonLog)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration from file if specified
	if *configFile != "" {
		cfg, err = config.New()
		if err != nil {
			logger.Fatal("Failed to load configuration", zap.Error(err))
		}
		logger.Info("Loaded configuration from file", zap.String("file", cfg.GetViper().ConfigFileUsed()))
	} else {
		// Create config from command line flags
		cfg = createConfigFromFlags()
	}

	// Initialize LLM client
	llmFactory := factory.NewLLMFactory(cfg, logger)
	llmClient, err := llmFactory.CreateLLMClient()
	if err != nil {
		logger.Fatal("Failed to create LLM client", zap.Error(err))
	}

	// Parse whitelisted domains
	var whitelistedDomains []string
	if *whitelistDomains != "" {
		whitelistedDomains = strings.Split(*whitelistDomains, ",")
		for i, domain := range whitelistedDomains {
			whitelistedDomains[i] = strings.TrimSpace(domain)
		}
	} else {
		whitelistedDomains = cfg.GetStringSlice("spam.whitelisted_domains")
	}
	
	if len(whitelistedDomains) > 0 {
		logger.Info("Using whitelisted domains", zap.Strings("domains", whitelistedDomains))
	}

	// Create whitelist checker
	whitelistChecker := whitelist.NewChecker(whitelistedDomains, logger)

	// Read email from file or stdin
	var emailReader io.Reader
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			logger.Fatal("Failed to open input file", zap.Error(err), zap.String("file", *inputFile))
		}
		defer file.Close()
		emailReader = file
		logger.Info("Reading email from file", zap.String("file", *inputFile))
	} else {
		emailReader = os.Stdin
		logger.Info("Reading email from stdin")
	}

	// Parse email
	msg, err := mail.ReadMessage(bufio.NewReader(emailReader))
	if err != nil {
		logger.Fatal("Failed to parse email", zap.Error(err))
	}

	// Extract email content
	from := msg.Header.Get("From")
	to := msg.Header.Get("To")
	subject := msg.Header.Get("Subject")

	// Read body
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		logger.Fatal("Failed to read email body", zap.Error(err))
	}
	body := string(bodyBytes)

	// Create email object
	email := &core.Email{
		From:    from,
		To:      strings.Split(to, ","),
		Subject: subject,
		Body:    body,
		Headers: make(map[string][]string),
	}

	// Copy headers
	for k, v := range msg.Header {
		email.Headers[k] = v
	}

	// Print email summary
	fmt.Printf("\n=== Email Summary ===\n")
	fmt.Printf("From: %s\n", from)
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Body length: %d bytes\n", len(body))
	fmt.Printf("\n")

	// Analyze email
	fmt.Printf("=== Analysis ===\n")
	fmt.Printf("Provider: %s\n", cfg.GetString("llm.provider"))
	fmt.Printf("Spam threshold: %.2f\n", cfg.GetFloat64("spam.threshold"))
	
	startTime := time.Now()
	
	// Check if sender domain is whitelisted
	if whitelistChecker.IsWhitelisted(from) {
		fmt.Printf("\n=== Results ===\n")
		fmt.Printf("Is spam: false (sender domain is whitelisted)\n")
		fmt.Printf("Spam score: 0.0\n")
		fmt.Printf("Confidence: 1.0\n")
		fmt.Printf("Explanation: Sender domain is whitelisted\n")
		fmt.Printf("Model used: whitelist\n")
		fmt.Printf("Processing time: %v\n", time.Since(startTime))
		return
	}
	
	result, err := llmClient.AnalyzeEmail(context.Background(), email)
	if err != nil {
		logger.Fatal("Failed to analyze email", zap.Error(err))
	}
	duration := time.Since(startTime)

	// Apply threshold
	result.IsSpam = result.Score >= cfg.GetFloat64("spam.threshold")

	// Print results
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Is spam: %t\n", result.IsSpam)
	fmt.Printf("Spam score: %.4f\n", result.Score)
	fmt.Printf("Confidence: %.4f\n", result.Confidence)
	fmt.Printf("Explanation: %s\n", result.Explanation)
	fmt.Printf("Model used: %s\n", result.ModelUsed)
	fmt.Printf("Processing time: %v\n", duration)
	
	// Close any resources that need closing
	if closer, ok := llmClient.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Failed to close LLM client", zap.Error(err))
		}
	}
}

// createConfigFromFlags creates a configuration from command line flags
func createConfigFromFlags() *config.Config {
	v := config.NewEmptyViper()
	
	// Set LLM provider
	v.Set("llm.provider", *provider)
	
	// Set provider-specific configuration
	switch *provider {
	case "bedrock":
		v.Set("bedrock.region", *bedrockRegion)
		v.Set("bedrock.model_id", *bedrockModelID)
		v.Set("bedrock.max_tokens", *maxTokens)
		v.Set("bedrock.temperature", *temperature)
		v.Set("bedrock.top_p", *topP)
		v.Set("bedrock.max_body_size", *maxBodySize)
	case "gemini":
		v.Set("gemini.api_key", *geminiAPIKey)
		v.Set("gemini.model_name", *geminiModelName)
		v.Set("gemini.max_tokens", *maxTokens)
		v.Set("gemini.temperature", *temperature)
		v.Set("gemini.top_p", *topP)
		v.Set("gemini.max_body_size", *maxBodySize)
	case "openai":
		v.Set("openai.api_key", *openaiAPIKey)
		v.Set("openai.model_name", *openaiModelName)
		v.Set("openai.max_tokens", *maxTokens)
		v.Set("openai.temperature", *temperature)
		v.Set("openai.top_p", *topP)
		v.Set("openai.max_body_size", *maxBodySize)
	}
	
	// Set spam threshold
	v.Set("spam.threshold", *spamThreshold)
	
	// Set whitelisted domains
	if *whitelistDomains != "" {
		domains := strings.Split(*whitelistDomains, ",")
		for i, domain := range domains {
			domains[i] = strings.TrimSpace(domain)
		}
		v.Set("spam.whitelisted_domains", domains)
	} else {
		v.Set("spam.whitelisted_domains", []string{})
	}
	
	return config.NewFromViper(v)
}
