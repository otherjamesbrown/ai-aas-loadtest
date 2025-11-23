package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

// Collector collects and exports metrics for load testing
type Collector struct {
	testRunID        string
	workerID         string
	pushgatewayURL   string
	pushInterval     time.Duration
	logger           *zap.Logger
	registry         *prometheus.Registry
	pusher           *push.Pusher

	// Worker metrics
	workerStatus       *prometheus.GaugeVec
	workerUsersActive  prometheus.Gauge

	// User/request metrics
	requestsTotal      *prometheus.CounterVec
	requestLatency     *prometheus.HistogramVec
	tokensTotal        *prometheus.CounterVec
	costUSD            *prometheus.CounterVec
	errorsTotal        *prometheus.CounterVec

	// LLM performance metrics
	llmTTFT            *prometheus.HistogramVec
	llmTPS             *prometheus.HistogramVec
	llmBackendSaturation *prometheus.GaugeVec
}

// NewCollector creates a new metrics collector
func NewCollector(testRunID, workerID, pushgatewayURL string, pushInterval time.Duration, logger *zap.Logger) *Collector {
	registry := prometheus.NewRegistry()

	// Worker metrics
	workerStatus := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "loadtest_worker_status",
			Help: "Worker lifecycle state (0=initializing, 1=bootstrapping, 2=running, 3=complete, 4=error)",
		},
		[]string{"test_run_id", "worker_id"},
	)

	workerUsersActive := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "loadtest_worker_users_active",
			Help: "Number of active user simulations in this worker",
		},
	)

	// User/request metrics
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadtest_requests_total",
			Help: "Total number of requests sent",
		},
		[]string{"test_run_id", "worker_id", "user_id", "org_id", "model", "success"},
	)

	requestLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loadtest_request_latency_seconds",
			Help:    "Request latency in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"test_run_id", "worker_id", "user_id", "org_id", "model"},
	)

	tokensTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadtest_tokens_total",
			Help: "Total tokens consumed",
		},
		[]string{"test_run_id", "worker_id", "user_id", "org_id", "model", "token_type"},
	)

	costUSD := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadtest_cost_usd",
			Help: "Estimated cost in USD",
		},
		[]string{"test_run_id", "worker_id", "user_id", "org_id", "model"},
	)

	errorsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadtest_errors_total",
			Help: "Total number of errors",
		},
		[]string{"test_run_id", "worker_id", "user_id", "org_id", "error_type"},
	)

	// LLM performance metrics
	llmTTFT := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loadtest_llm_time_to_first_token_milliseconds",
			Help:    "Time from request to first token received (TTFT)",
			Buckets: []float64{50, 100, 200, 300, 500, 750, 1000, 1500, 2000, 3000, 5000, 10000},
		},
		[]string{"test_run_id", "worker_id", "user_id", "model"},
	)

	llmTPS := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loadtest_llm_tokens_per_second",
			Help:    "Tokens generated per second (TPS)",
			Buckets: []float64{1, 5, 10, 20, 30, 40, 50, 75, 100, 150, 200},
		},
		[]string{"test_run_id", "worker_id", "user_id", "model"},
	)

	llmBackendSaturation := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "loadtest_llm_backend_saturation_ratio",
			Help: "Backend saturation ratio (0.0-1.0)",
		},
		[]string{"test_run_id", "worker_id", "model"},
	)

	// Register all metrics
	registry.MustRegister(
		workerStatus,
		workerUsersActive,
		requestsTotal,
		requestLatency,
		tokensTotal,
		costUSD,
		errorsTotal,
		llmTTFT,
		llmTPS,
		llmBackendSaturation,
	)

	// Create pusher
	pusher := push.New(pushgatewayURL, "loadtest_worker").
		Gatherer(registry).
		Grouping("test_run_id", testRunID).
		Grouping("worker_id", workerID)

	return &Collector{
		testRunID:            testRunID,
		workerID:             workerID,
		pushgatewayURL:       pushgatewayURL,
		pushInterval:         pushInterval,
		logger:               logger,
		registry:             registry,
		pusher:               pusher,
		workerStatus:         workerStatus,
		workerUsersActive:    workerUsersActive,
		requestsTotal:        requestsTotal,
		requestLatency:       requestLatency,
		tokensTotal:          tokensTotal,
		costUSD:              costUSD,
		errorsTotal:          errorsTotal,
		llmTTFT:              llmTTFT,
		llmTPS:               llmTPS,
		llmBackendSaturation: llmBackendSaturation,
	}
}

// SetWorkerStatus sets the worker status
func (c *Collector) SetWorkerStatus(status int) {
	c.workerStatus.WithLabelValues(c.testRunID, c.workerID).Set(float64(status))
}

// SetActiveUsers sets the number of active user simulations
func (c *Collector) SetActiveUsers(count int) {
	c.workerUsersActive.Set(float64(count))
}

// RecordRequest records a completed request with its metrics
func (c *Collector) RecordRequest(userID, orgID, model string, latency time.Duration, tokens int, success bool, errorType string) {
	successStr := "true"
	if !success {
		successStr = "false"
	}

	// Increment request counter
	c.requestsTotal.WithLabelValues(
		c.testRunID,
		c.workerID,
		userID,
		orgID,
		model,
		successStr,
	).Inc()

	if success {
		// Record latency
		c.requestLatency.WithLabelValues(
			c.testRunID,
			c.workerID,
			userID,
			orgID,
			model,
		).Observe(latency.Seconds())

		// Record tokens
		if tokens > 0 {
			c.tokensTotal.WithLabelValues(
				c.testRunID,
				c.workerID,
				userID,
				orgID,
				model,
				"total",
			).Add(float64(tokens))

			// Estimate cost (simplified pricing)
			cost := c.estimateCost(model, tokens)
			if cost > 0 {
				c.costUSD.WithLabelValues(
					c.testRunID,
					c.workerID,
					userID,
					orgID,
					model,
				).Add(cost)
			}
		}
	} else {
		// Record error
		c.errorsTotal.WithLabelValues(
			c.testRunID,
			c.workerID,
			userID,
			orgID,
			errorType,
		).Inc()
	}
}

// RecordLLMMetrics records LLM-specific performance metrics
func (c *Collector) RecordLLMMetrics(userID, model string, ttft time.Duration, tps float64) {
	// Record TTFT in milliseconds
	c.llmTTFT.WithLabelValues(
		c.testRunID,
		c.workerID,
		userID,
		model,
	).Observe(float64(ttft.Milliseconds()))

	// Record TPS
	c.llmTPS.WithLabelValues(
		c.testRunID,
		c.workerID,
		userID,
		model,
	).Observe(tps)
}

// SetBackendSaturation sets the backend saturation ratio
func (c *Collector) SetBackendSaturation(model string, ratio float64) {
	c.llmBackendSaturation.WithLabelValues(
		c.testRunID,
		c.workerID,
		model,
	).Set(ratio)
}

// estimateCost estimates the cost of a request based on model and tokens
// This is a simplified estimation - actual costs vary by model and token type
func (c *Collector) estimateCost(model string, tokens int) float64 {
	// Simplified pricing (USD per 1M tokens)
	// In reality, prompt and completion tokens have different prices
	pricePerMillionTokens := map[string]float64{
		"gpt-4o":          5.0,   // Average of input/output
		"gpt-4":           30.0,  // Average
		"gpt-3.5-turbo":   0.5,   // Average
		"claude-3-opus":   15.0,  // Average
		"claude-3-sonnet": 3.0,   // Average
	}

	price, exists := pricePerMillionTokens[model]
	if !exists {
		price = 5.0 // Default price
	}

	return (float64(tokens) / 1_000_000.0) * price
}

// Push pushes metrics to the Pushgateway
func (c *Collector) Push() error {
	if err := c.pusher.Push(); err != nil {
		c.logger.Error("Failed to push metrics",
			zap.String("pushgateway", c.pushgatewayURL),
			zap.Error(err),
		)
		return fmt.Errorf("failed to push metrics: %w", err)
	}

	c.logger.Debug("Metrics pushed successfully",
		zap.String("test_run_id", c.testRunID),
		zap.String("worker_id", c.workerID),
	)

	return nil
}

// StartPeriodicPush starts pushing metrics periodically in a background goroutine
func (c *Collector) StartPeriodicPush(stopCh <-chan struct{}) {
	ticker := time.NewTicker(c.pushInterval)
	defer ticker.Stop()

	c.logger.Info("Started periodic metrics push",
		zap.Duration("interval", c.pushInterval),
		zap.String("pushgateway", c.pushgatewayURL),
	)

	for {
		select {
		case <-ticker.C:
			if err := c.Push(); err != nil {
				c.logger.Error("Periodic push failed", zap.Error(err))
			}
		case <-stopCh:
			c.logger.Info("Stopping periodic metrics push")
			// Final push before stopping
			if err := c.Push(); err != nil {
				c.logger.Error("Final push failed", zap.Error(err))
			}
			return
		}
	}
}

// Delete deletes metrics from the Pushgateway (cleanup)
func (c *Collector) Delete() error {
	if err := c.pusher.Delete(); err != nil {
		return fmt.Errorf("failed to delete metrics: %w", err)
	}
	return nil
}
