package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/otherjamesbrown/ai-aas-loadtest/internal/config"
	"github.com/otherjamesbrown/ai-aas-loadtest/internal/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	configPath      = flag.String("config", "", "Path to load test configuration YAML file")
	pushgatewayURL  = flag.String("pushgateway", "http://localhost:9091", "Prometheus Pushgateway URL")
	logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	version         = "dev" // Set during build
)

func main() {
	flag.Parse()

	// Setup logger
	logger, err := setupLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Load Test Worker starting",
		zap.String("version", version),
	)

	// Validate flags
	if *configPath == "" {
		// Try environment variable
		*configPath = os.Getenv("CONFIG_PATH")
		if *configPath == "" {
			logger.Fatal("Config path is required (use --config or CONFIG_PATH env var)")
		}
	}

	// Load configuration
	logger.Info("Loading configuration", zap.String("path", *configPath))
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configPath),
			zap.Error(err),
		)
	}

	logger.Info("Configuration loaded successfully",
		zap.String("test_name", cfg.Metadata.Name),
		zap.Int("organizations", cfg.Spec.Organizations.Count),
		zap.Int("users_min", cfg.Spec.Users.PerOrg.Min),
		zap.Int("users_max", cfg.Spec.Users.PerOrg.Max),
	)

	// Get Pushgateway URL from environment if set
	if pgURL := os.Getenv("PUSHGATEWAY_URL"); pgURL != "" {
		*pushgatewayURL = pgURL
	}

	// Create worker
	w, err := worker.NewWorker(cfg, logger, *pushgatewayURL)
	if err != nil {
		logger.Fatal("Failed to create worker", zap.Error(err))
	}

	logger.Info("Worker created",
		zap.String("test_run_id", w.GetTestRunID()),
		zap.String("worker_id", w.GetWorkerID()),
	)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Warn("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Run the load test
	logger.Info("Starting load test execution")
	if err := w.Run(ctx); err != nil {
		logger.Error("Load test failed", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Load test completed successfully",
		zap.String("test_run_id", w.GetTestRunID()),
	)
}

// setupLogger creates a zap logger with the specified log level
func setupLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       false,
		Encoding:          "json",
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     false,
		DisableStacktrace: false,
	}

	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}
