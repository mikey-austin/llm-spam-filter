package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mikey/llm-spam-filter/internal/core"
	"github.com/mikey/llm-spam-filter/internal/di"
	"github.com/mikey/llm-spam-filter/internal/ports"
	"go.uber.org/zap"
)

func main() {
	// Build the dependency injection container
	container, err := di.BuildContainer()
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
	cacheRepo core.CacheRepository,
) error {
	defer logger.Sync()

	// Start the filter
	if err := emailFilter.Start(); err != nil {
		logger.Fatal("Failed to start filter", zap.Error(err))
		return err
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Info("Shutting down...")

	// Stop the filter
	if err := emailFilter.Stop(); err != nil {
		logger.Error("Failed to stop filter", zap.Error(err))
	}

	// Close any resources that need closing
	if closer, ok := llmClient.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Error("Failed to close LLM client", zap.Error(err))
		}
	}

	// Stop the cache if needed
	if stopper, ok := cacheRepo.(interface{ Stop() }); ok {
		stopper.Stop()
	}

	logger.Info("Shutdown complete")
	return nil
}
