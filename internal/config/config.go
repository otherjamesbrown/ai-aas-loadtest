package config

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// LoadFromFile loads a LoadTestConfig from a YAML file.
func LoadFromFile(path string) (*LoadTestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return LoadFromBytes(data)
}

// LoadFromBytes loads a LoadTestConfig from YAML bytes.
func LoadFromBytes(data []byte) (*LoadTestConfig, error) {
	var config LoadTestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &config, nil
}

// Validate validates a LoadTestConfig for required fields and logical consistency.
func Validate(config *LoadTestConfig) error {
	// Validate apiVersion and kind
	if config.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if config.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	// Validate metadata
	if config.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Validate organizations
	if config.Spec.Organizations.Count < 1 {
		return fmt.Errorf("organizations.count must be >= 1")
	}

	// Validate users
	if config.Spec.Users.PerOrg.Min < 1 {
		return fmt.Errorf("users.perOrg.min must be >= 1")
	}
	if config.Spec.Users.PerOrg.Max < config.Spec.Users.PerOrg.Min {
		return fmt.Errorf("users.perOrg.max must be >= users.perOrg.min")
	}

	// Validate think time
	if config.Spec.UserBehavior.ThinkTimeSeconds.Base < 0 {
		return fmt.Errorf("userBehavior.thinkTimeSeconds.base must be >= 0")
	}
	if config.Spec.UserBehavior.ThinkTimeSeconds.Variance < 0 {
		return fmt.Errorf("userBehavior.thinkTimeSeconds.variance must be >= 0")
	}

	// Validate think time distribution
	if config.Spec.UserBehavior.ThinkTimeSeconds.Distribution != "" {
		valid := map[string]bool{
			"uniform":     true,
			"gaussian":    true,
			"exponential": true,
		}
		if !valid[config.Spec.UserBehavior.ThinkTimeSeconds.Distribution] {
			return fmt.Errorf("userBehavior.thinkTimeSeconds.distribution must be one of: uniform, gaussian, exponential")
		}
	}

	// Apply defaults for think time
	if config.Spec.UserBehavior.ThinkTimeSeconds.Distribution == "" {
		config.Spec.UserBehavior.ThinkTimeSeconds.Distribution = "uniform"
	}
	if config.Spec.UserBehavior.ThinkTimeSeconds.Min == 0 {
		config.Spec.UserBehavior.ThinkTimeSeconds.Min = 1 // Default min 1 second
	}
	if config.Spec.UserBehavior.ThinkTimeSeconds.Max == 0 {
		config.Spec.UserBehavior.ThinkTimeSeconds.Max = 60 // Default max 60 seconds
	}

	// Validate targets
	if config.Spec.Targets.APIRouterURL == "" {
		return fmt.Errorf("targets.apiRouterUrl is required")
	}
	if config.Spec.Targets.UserOrgURL == "" {
		return fmt.Errorf("targets.userOrgUrl is required")
	}

	// Validate scenario mix weights sum to 100 if specified
	if len(config.Spec.Scenarios.Scenarios) > 0 {
		totalWeight := 0
		for _, s := range config.Spec.Scenarios.Scenarios {
			if s.Weight < 0 {
				return fmt.Errorf("scenario weight must be >= 0: %s", s.Name)
			}
			totalWeight += s.Weight
		}
		if totalWeight != 100 {
			return fmt.Errorf("scenario weights must sum to 100, got %d", totalWeight)
		}
	}

	// Validate test type weights sum to 100 if specified
	if len(config.Spec.TestTypes) > 0 {
		totalWeight := 0
		for _, tt := range config.Spec.TestTypes {
			if tt.Weight < 0 {
				return fmt.Errorf("test type weight must be >= 0: %s", tt.Name)
			}
			totalWeight += tt.Weight
		}
		if totalWeight != 100 {
			return fmt.Errorf("test type weights must sum to 100, got %d", totalWeight)
		}
	}

	// Validate limits
	if config.Spec.Limits.MaxErrorRate < 0 || config.Spec.Limits.MaxErrorRate > 1 {
		return fmt.Errorf("limits.maxErrorRate must be between 0 and 1")
	}

	return nil
}

// NewRuntimeConfig creates a new RuntimeConfig with generated IDs and current timestamp.
func NewRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		TestRunID:     generateTestRunID(),
		WorkerID:      generateWorkerID(),
		StartTime:     time.Now(),
		Organizations: []BootstrappedOrg{},
		Users:         []BootstrappedUser{},
		APIKeys:       make(map[string]string),
	}
}

// generateTestRunID generates a unique test run ID.
// Format: test-<timestamp>-<short-uuid>
func generateTestRunID() string {
	timestamp := time.Now().Format("20060102-150405")
	shortUUID := uuid.New().String()[:8]
	return fmt.Sprintf("test-%s-%s", timestamp, shortUUID)
}

// generateWorkerID generates a unique worker ID.
// Format: worker-<hostname>-<short-uuid>
func generateWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	shortUUID := uuid.New().String()[:8]
	return fmt.Sprintf("worker-%s-%s", hostname, shortUUID)
}

// GetThinkTime calculates a think time duration based on the configuration.
// Implements the "base ± variance" pattern with different distribution types.
func (tc *ThinkTimeConfig) GetThinkTime(rng RandomNumberGenerator) time.Duration {
	var seconds float64

	switch tc.Distribution {
	case "gaussian":
		// Gaussian distribution: mean = base, stddev = variance/2
		// This gives ~95% of values within base ± variance
		stddev := float64(tc.Variance) / 2.0
		seconds = rng.NormalFloat64(float64(tc.Base), stddev)

	case "exponential":
		// Exponential distribution with mean = base
		// Note: This is right-skewed, most values will be < base
		seconds = rng.ExpFloat64(float64(tc.Base))

	default: // "uniform" or empty
		// Uniform distribution: [base - variance, base + variance]
		minVal := float64(tc.Base - tc.Variance)
		maxVal := float64(tc.Base + tc.Variance)
		seconds = rng.UniformFloat64(minVal, maxVal)
	}

	// Clamp to min/max bounds
	if seconds < float64(tc.Min) {
		seconds = float64(tc.Min)
	}
	if seconds > float64(tc.Max) {
		seconds = float64(tc.Max)
	}

	return time.Duration(seconds * float64(time.Second))
}

// RandomNumberGenerator defines the interface for random number generation.
// This allows for seeded randomness for reproducible tests.
type RandomNumberGenerator interface {
	// UniformFloat64 returns a uniformly distributed float64 in [min, max]
	UniformFloat64(min, max float64) float64

	// NormalFloat64 returns a normally distributed float64 with given mean and stddev
	NormalFloat64(mean, stddev float64) float64

	// ExpFloat64 returns an exponentially distributed float64 with given mean
	ExpFloat64(mean float64) float64

	// Intn returns a random int in [0, n)
	Intn(n int) int
}

// DefaultRNG is a default implementation using math/rand
type DefaultRNG struct {
	rng interface {
		Float64() float64
		NormFloat64() float64
		ExpFloat64() float64
		Intn(n int) int
	}
}

// NewDefaultRNG creates a new DefaultRNG with the given seed source
func NewDefaultRNG(rng interface {
	Float64() float64
	NormFloat64() float64
	ExpFloat64() float64
	Intn(n int) int
}) *DefaultRNG {
	return &DefaultRNG{rng: rng}
}

func (r *DefaultRNG) UniformFloat64(min, max float64) float64 {
	return min + r.rng.Float64()*(max-min)
}

func (r *DefaultRNG) NormalFloat64(mean, stddev float64) float64 {
	return mean + r.rng.NormFloat64()*stddev
}

func (r *DefaultRNG) ExpFloat64(mean float64) float64 {
	return r.rng.ExpFloat64() * mean
}

func (r *DefaultRNG) Intn(n int) int {
	return r.rng.Intn(n)
}

// GetUserCount calculates a random number of users within the configured range.
func (uc *UserConfig) GetUserCount(rng RandomNumberGenerator) int {
	if uc.PerOrg.Min == uc.PerOrg.Max {
		return uc.PerOrg.Min
	}
	return uc.PerOrg.Min + rng.Intn(uc.PerOrg.Max-uc.PerOrg.Min+1)
}

// GetSessionDuration calculates a random session duration within the configured range.
func (dc *DurationConfig) GetDuration(rng RandomNumberGenerator) time.Duration {
	if dc.Min == dc.Max {
		return time.Duration(dc.Min) * time.Second
	}
	seconds := dc.Min + rng.Intn(dc.Max-dc.Min+1)
	return time.Duration(seconds) * time.Second
}

// GetQuestionCount calculates a random number of questions within the configured range.
func (rc *RangeConfig) GetValue(rng RandomNumberGenerator) int {
	if rc.Min == rc.Max {
		return rc.Min
	}
	return rc.Min + rng.Intn(rc.Max-rc.Min+1)
}
