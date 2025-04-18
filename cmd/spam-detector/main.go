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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/mikey/llm-spam-filter/internal/adapters/bedrock"
	"github.com/mikey/llm-spam-filter/internal/adapters/gemini"
	"github.com/mikey/llm-spam-filter/internal/adapters/openai"
	"github.com/mikey/llm-spam-filter/internal/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := initLogger(*verbose, *jsonLog)
	defer logger.Sync()

	// Initialize LLM client
	llmClient, err := createLLMClient(logger)
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
		logger.Info("Using whitelisted domains", zap.Strings("domains", whitelistedDomains))
	}

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
	fmt.Printf("Provider: %s\n", *provider)
	fmt.Printf("Spam threshold: %.2f\n", *spamThreshold)
	
	startTime := time.Now()
	
	// Check if sender domain is whitelisted
	if isWhitelisted(from, whitelistedDomains) {
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
	result.IsSpam = result.Score >= *spamThreshold

	// Print results
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Is spam: %t\n", result.IsSpam)
	fmt.Printf("Spam score: %.4f\n", result.Score)
	fmt.Printf("Confidence: %.4f\n", result.Confidence)
	fmt.Printf("Explanation: %s\n", result.Explanation)
	fmt.Printf("Model used: %s\n", result.ModelUsed)
	fmt.Printf("Processing time: %v\n", duration)
	
	// Close Gemini client if needed
	if geminiClient, ok := llmClient.(*gemini.GeminiClient); ok {
		if err := geminiClient.Close(); err != nil {
			logger.Error("Failed to close Gemini client", zap.Error(err))
		}
	}
}

func createLLMClient(logger *zap.Logger) (core.LLMClient, error) {
	switch *provider {
	case "bedrock":
		// Initialize AWS client
		awsCfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(*bedrockRegion),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
		}

		// Initialize Bedrock client
		bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
		return bedrock.NewBedrockClient(
			bedrockClient,
			*bedrockModelID,
			*maxTokens,
			float32(*temperature),
			float32(*topP),
			*maxBodySize,
			logger,
		), nil

	case "gemini":
		if *geminiAPIKey == "" {
			return nil, fmt.Errorf("gemini API key is required")
		}
		
		// Initialize Gemini client
		factory := gemini.NewFactory(
			*geminiAPIKey,
			*geminiModelName,
			*maxTokens,
			float32(*temperature),
			float32(*topP),
			*maxBodySize,
			logger,
		)
		return factory.CreateLLMClient()

	case "openai":
		if *openaiAPIKey == "" {
			return nil, fmt.Errorf("openai API key is required")
		}
		
		// Initialize OpenAI client
		factory := openai.NewFactory(
			*openaiAPIKey,
			*openaiModelName,
			*maxTokens,
			float32(*temperature),
			float32(*topP),
			*maxBodySize,
			logger,
		)
		return factory.CreateLLMClient()

	default:
		return nil, fmt.Errorf("invalid LLM provider: %s", *provider)
	}
}

// isWhitelisted checks if the sender's domain is in the whitelist
func isWhitelisted(from string, whitelistedDomains []string) bool {
	if len(whitelistedDomains) == 0 {
		return false
	}

	// Extract domain from email address
	parts := strings.Split(from, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])

	// Check if domain is in whitelist
	for _, whitelisted := range whitelistedDomains {
		if strings.ToLower(whitelisted) == domain {
			return true
		}
	}

	return false
}

func initLogger(verbose bool, jsonFormat bool) *zap.Logger {
	var level zapcore.Level
	if verbose {
		level = zapcore.DebugLevel
	} else {
		level = zapcore.InfoLevel
	}

	var config zap.Config
	if jsonFormat {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	config.Level = zap.NewAtomicLevelAt(level)

	logger, err := config.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}
