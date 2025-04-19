package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"

	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/di"
	"github.com/mikey/llm-spam-filter/internal/ports"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	flags := di.ParseFlags()

	// Build the dependency injection container
	container, err := di.BuildCLIContainer(flags)
	if err != nil {
		fmt.Printf("Failed to build dependency container: %v\n", err)
		os.Exit(1)
	}

	// Run the application
	if err := container.Invoke(run); err != nil {
		fmt.Printf("Application error: %v\n", err)
		os.Exit(1)
	}
}

// run is the main application function that gets all dependencies injected
func run(
	logger *zap.Logger,
	emailFilter ports.EmailFilter,
	llmClient core.LLMClient,
	flags *di.CLIFlags,
) error {
	defer logger.Sync()

	// Read email from file or stdin
	email := readEmail(logger, flags.InputFile)

	// Process the email
	ctx := context.Background()
	_, err := emailFilter.ProcessEmail(ctx, email)
	if err != nil {
		logger.Error("Failed to process email", zap.Error(err))
		return err
	}

	// Close any resources that need closing
	if closer, ok := llmClient.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Failed to close LLM client", zap.Error(err))
		}
	}

	return nil
}

// readEmail reads an email from a file or stdin
func readEmail(logger *zap.Logger, inputFile string) *core.Email {
	// Read email from file or stdin
	var emailReader io.Reader
	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			logger.Fatal("Failed to open input file", zap.Error(err), zap.String("file", inputFile))
		}
		defer file.Close()
		emailReader = file
		logger.Info("Reading email from file", zap.String("file", inputFile))
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

	return email
}
