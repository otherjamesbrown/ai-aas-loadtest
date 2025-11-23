package worker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/otherjamesbrown/ai-aas-loadtest/internal/bootstrap"
	"github.com/otherjamesbrown/ai-aas-loadtest/internal/config"
	"github.com/otherjamesbrown/ai-aas-loadtest/internal/metrics"
	"github.com/otherjamesbrown/ai-aas-loadtest/internal/simulator"
	"go.uber.org/zap"
)

// Worker orchestrates the entire load test execution
type Worker struct {
	cfg             *config.LoadTestConfig
	runtime         *config.RuntimeConfig
	logger          *zap.Logger
	metrics         *metrics.Collector
	bootstrapMgr    *bootstrap.Manager
	stopMetricsCh   chan struct{}
}

// WorkerStatus represents the worker's current state
type WorkerStatus int

const (
	StatusInitializing WorkerStatus = 0
	StatusBootstrapping WorkerStatus = 1
	StatusRunning      WorkerStatus = 2
	StatusComplete     WorkerStatus = 3
	StatusError        WorkerStatus = 4
)

// NewWorker creates a new load test worker
func NewWorker(cfg *config.LoadTestConfig, logger *zap.Logger, pushgatewayURL string) (*Worker, error) {
	runtime := config.NewRuntimeConfig()

	// Create metrics collector
	metricsCollector := metrics.NewCollector(
		runtime.TestRunID,
		runtime.WorkerID,
		pushgatewayURL,
		10*time.Second, // Push every 10 seconds
		logger,
	)

	// Create bootstrap manager
	timeout := 60 * time.Second
	if cfg.Spec.Targets.Timeout > 0 {
		timeout = time.Duration(cfg.Spec.Targets.Timeout) * time.Second
	}

	bootstrapMgr := bootstrap.NewManager(
		cfg.Spec.Targets.UserOrgURL,
		cfg.Spec.Targets.AdminAPIKey,
		timeout,
		logger,
	)

	return &Worker{
		cfg:           cfg,
		runtime:       runtime,
		logger:        logger,
		metrics:       metricsCollector,
		bootstrapMgr:  bootstrapMgr,
		stopMetricsCh: make(chan struct{}),
	}, nil
}

// Run executes the complete load test workflow
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("Load test worker starting",
		zap.String("test_run_id", w.runtime.TestRunID),
		zap.String("worker_id", w.runtime.WorkerID),
		zap.String("test_name", w.cfg.Metadata.Name),
	)

	// Phase 1: Initialization
	w.metrics.SetWorkerStatus(int(StatusInitializing))
	w.logger.Info("Phase 1: Initialization")

	// Start periodic metrics push
	go w.metrics.StartPeriodicPush(w.stopMetricsCh)

	// Phase 2: Bootstrap
	w.metrics.SetWorkerStatus(int(StatusBootstrapping))
	w.logger.Info("Phase 2: Bootstrapping organizations, users, and API keys")

	if err := w.bootstrapMgr.Bootstrap(w.cfg, w.runtime); err != nil {
		w.metrics.SetWorkerStatus(int(StatusError))
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	w.logger.Info("Bootstrap complete",
		zap.Int("organizations", len(w.runtime.Organizations)),
		zap.Int("users", len(w.runtime.Users)),
		zap.Int("api_keys", len(w.runtime.APIKeys)),
	)

	// Phase 3: Run simulations
	w.metrics.SetWorkerStatus(int(StatusRunning))
	w.logger.Info("Phase 3: Running user simulations")

	if err := w.runSimulations(ctx); err != nil {
		w.metrics.SetWorkerStatus(int(StatusError))
		return fmt.Errorf("simulations failed: %w", err)
	}

	// Phase 4: Cleanup and completion
	w.metrics.SetWorkerStatus(int(StatusComplete))
	w.logger.Info("Phase 4: Cleanup and completion")

	// Stop periodic metrics push
	close(w.stopMetricsCh)

	// Optional: Clean up created resources
	if shouldCleanup() {
		w.logger.Info("Cleaning up test resources")
		if err := w.bootstrapMgr.Cleanup(w.runtime); err != nil {
			w.logger.Error("Cleanup failed", zap.Error(err))
			// Don't fail the test on cleanup errors
		}
	}

	w.logger.Info("Load test complete",
		zap.String("test_run_id", w.runtime.TestRunID),
		zap.Duration("duration", time.Since(w.runtime.StartTime)),
	)

	return nil
}

// runSimulations runs all user simulations concurrently
func (w *Worker) runSimulations(ctx context.Context) error {
	// Create RNG for consistent randomness
	baseRNG := rand.New(rand.NewSource(time.Now().UnixNano()))
	rngConfig := config.NewDefaultRNG(baseRNG)

	// Set active users metric
	w.metrics.SetActiveUsers(len(w.runtime.Users))

	// Create wait group for all user goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, len(w.runtime.Users))

	// Create context with timeout if specified
	simCtx := ctx
	if w.cfg.Spec.Limits.MaxDurationSec > 0 {
		var cancel context.CancelFunc
		simCtx, cancel = context.WithTimeout(ctx, time.Duration(w.cfg.Spec.Limits.MaxDurationSec)*time.Second)
		defer cancel()
	}

	// Start simulation for each user concurrently
	for i, user := range w.runtime.Users {
		apiKey, exists := w.runtime.APIKeys[user.ID]
		if !exists {
			w.logger.Error("No API key found for user",
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
			)
			continue
		}

		wg.Add(1)

		// Create unique seed for this user
		userSeed := int64(i * 1337)

		// Get timeout for HTTP requests
		timeout := 60 * time.Second
		if w.cfg.Spec.Targets.Timeout > 0 {
			timeout = time.Duration(w.cfg.Spec.Targets.Timeout) * time.Second
		}

		// Create user simulator
		sim := simulator.NewUserSimulator(
			&user,
			apiKey,
			w.cfg.Spec.Targets.APIRouterURL,
			userSeed,
			timeout,
			w.logger,
			w.metrics,
		)

		// Run simulation in goroutine
		go func(sim *simulator.UserSimulator, user config.BootstrappedUser) {
			defer wg.Done()

			w.logger.Info("Starting user simulation",
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
				zap.String("org_name", user.OrgName),
			)

			if err := sim.Run(simCtx, w.cfg, rngConfig); err != nil {
				w.logger.Error("User simulation failed",
					zap.String("user_id", user.ID),
					zap.Error(err),
				)
				select {
				case errChan <- err:
				default:
				}
			}

			w.logger.Info("User simulation complete",
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
			)
		}(sim, user)

		// Small stagger between starting users to avoid thundering herd
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all simulations to complete
	w.logger.Info("Waiting for all user simulations to complete",
		zap.Int("user_count", len(w.runtime.Users)),
	)

	wg.Wait()
	close(errChan)

	// Check if any simulations failed
	var firstError error
	errorCount := 0
	for err := range errChan {
		if err != nil {
			errorCount++
			if firstError == nil {
				firstError = err
			}
		}
	}

	if errorCount > 0 {
		w.logger.Warn("Some user simulations failed",
			zap.Int("error_count", errorCount),
			zap.Int("total_users", len(w.runtime.Users)),
		)
		// Don't fail the entire test if only some users failed
		// Return error only if majority failed
		if errorCount > len(w.runtime.Users)/2 {
			return fmt.Errorf("majority of simulations failed: %w", firstError)
		}
	}

	return nil
}

// shouldCleanup determines if resources should be cleaned up after the test
func shouldCleanup() bool {
	// For now, don't cleanup to allow inspection
	// In production, this could be controlled by config or env var
	return false
}

// GetTestRunID returns the test run ID
func (w *Worker) GetTestRunID() string {
	return w.runtime.TestRunID
}

// GetWorkerID returns the worker ID
func (w *Worker) GetWorkerID() string {
	return w.runtime.WorkerID
}

// GetMetrics returns the metrics collector
func (w *Worker) GetMetrics() *metrics.Collector {
	return w.metrics
}
