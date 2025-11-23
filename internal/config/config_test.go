package config

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromBytes(t *testing.T) {
	validYAML := `
apiVersion: loadtest.ai-aas.dev/v1
kind: LoadTestScenario
metadata:
  name: test-scenario
  namespace: load-testing
spec:
  organizations:
    count: 1
  users:
    perOrg:
      min: 10
      max: 50
  userBehavior:
    thinkTimeSeconds:
      base: 5
      variance: 2
  targets:
    apiRouterUrl: https://api.example.com
    userOrgUrl: https://admin.example.com
`

	config, err := LoadFromBytes([]byte(validYAML))
	require.NoError(t, err)
	assert.Equal(t, "loadtest.ai-aas.dev/v1", config.APIVersion)
	assert.Equal(t, "LoadTestScenario", config.Kind)
	assert.Equal(t, "test-scenario", config.Metadata.Name)
	assert.Equal(t, 1, config.Spec.Organizations.Count)
	assert.Equal(t, 10, config.Spec.Users.PerOrg.Min)
	assert.Equal(t, 50, config.Spec.Users.PerOrg.Max)
	assert.Equal(t, 5, config.Spec.UserBehavior.ThinkTimeSeconds.Base)
	assert.Equal(t, 2, config.Spec.UserBehavior.ThinkTimeSeconds.Variance)
	assert.Equal(t, "uniform", config.Spec.UserBehavior.ThinkTimeSeconds.Distribution) // Default
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		config      LoadTestConfig
		expectedErr string
	}{
		{
			name: "missing apiVersion",
			config: LoadTestConfig{
				Kind: "LoadTestScenario",
			},
			expectedErr: "apiVersion is required",
		},
		{
			name: "missing kind",
			config: LoadTestConfig{
				APIVersion: "loadtest.ai-aas.dev/v1",
			},
			expectedErr: "kind is required",
		},
		{
			name: "missing metadata.name",
			config: LoadTestConfig{
				APIVersion: "loadtest.ai-aas.dev/v1",
				Kind:       "LoadTestScenario",
			},
			expectedErr: "metadata.name is required",
		},
		{
			name: "zero organizations",
			config: LoadTestConfig{
				APIVersion: "loadtest.ai-aas.dev/v1",
				Kind:       "LoadTestScenario",
				Metadata:   LoadTestMetadata{Name: "test"},
				Spec: LoadTestSpec{
					Organizations: OrganizationConfig{Count: 0},
				},
			},
			expectedErr: "organizations.count must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestValidate_ThinkTimeDistribution(t *testing.T) {
	config := LoadTestConfig{
		APIVersion: "loadtest.ai-aas.dev/v1",
		Kind:       "LoadTestScenario",
		Metadata:   LoadTestMetadata{Name: "test"},
		Spec: LoadTestSpec{
			Organizations: OrganizationConfig{Count: 1},
			Users:         UserConfig{PerOrg: RangeConfig{Min: 1, Max: 10}},
			UserBehavior: BehaviorConfig{
				ThinkTimeSeconds: ThinkTimeConfig{
					Base:         5,
					Variance:     2,
					Distribution: "invalid",
				},
			},
			Targets: TargetConfig{
				APIRouterURL: "https://api.example.com",
				UserOrgURL:   "https://admin.example.com",
			},
		},
	}

	err := Validate(&config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "distribution must be one of")
}

func TestValidate_ScenarioWeights(t *testing.T) {
	config := LoadTestConfig{
		APIVersion: "loadtest.ai-aas.dev/v1",
		Kind:       "LoadTestScenario",
		Metadata:   LoadTestMetadata{Name: "test"},
		Spec: LoadTestSpec{
			Organizations: OrganizationConfig{Count: 1},
			Users:         UserConfig{PerOrg: RangeConfig{Min: 1, Max: 10}},
			UserBehavior: BehaviorConfig{
				ThinkTimeSeconds: ThinkTimeConfig{Base: 5, Variance: 2},
			},
			Targets: TargetConfig{
				APIRouterURL: "https://api.example.com",
				UserOrgURL:   "https://admin.example.com",
			},
			Scenarios: ScenarioMixConfig{
				Scenarios: []ScenarioWeight{
					{Name: "scenario1", Weight: 75},
					{Name: "scenario2", Weight: 20}, // Sum = 95, should fail
				},
			},
		},
	}

	err := Validate(&config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scenario weights must sum to 100")
}

func TestNewRuntimeConfig(t *testing.T) {
	runtime := NewRuntimeConfig()

	assert.NotEmpty(t, runtime.TestRunID)
	assert.NotEmpty(t, runtime.WorkerID)
	assert.True(t, time.Since(runtime.StartTime) < 1*time.Second)
	assert.NotNil(t, runtime.Organizations)
	assert.NotNil(t, runtime.Users)
	assert.NotNil(t, runtime.APIKeys)
}

func TestThinkTimeConfig_GetThinkTime_Uniform(t *testing.T) {
	config := ThinkTimeConfig{
		Base:         5,
		Variance:     2,
		Distribution: "uniform",
		Min:          1,
		Max:          30,
	}

	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples to ensure they're within bounds
	for i := 0; i < 100; i++ {
		duration := config.GetThinkTime(rng)
		seconds := duration.Seconds()

		// Uniform should be between base - variance and base + variance
		// With clamping to min/max
		assert.GreaterOrEqual(t, seconds, float64(config.Min))
		assert.LessOrEqual(t, seconds, float64(config.Max))
	}
}

func TestThinkTimeConfig_GetThinkTime_Gaussian(t *testing.T) {
	config := ThinkTimeConfig{
		Base:         10,
		Variance:     4,
		Distribution: "gaussian",
		Min:          1,
		Max:          30,
	}

	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples
	samples := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		duration := config.GetThinkTime(rng)
		seconds := duration.Seconds()
		samples[i] = seconds

		// Should be within min/max bounds
		assert.GreaterOrEqual(t, seconds, float64(config.Min))
		assert.LessOrEqual(t, seconds, float64(config.Max))
	}

	// Calculate mean (should be close to base)
	sum := 0.0
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))

	// Mean should be reasonably close to base (within 1 second for 1000 samples)
	assert.InDelta(t, float64(config.Base), mean, 1.0)
}

func TestThinkTimeConfig_GetThinkTime_Exponential(t *testing.T) {
	config := ThinkTimeConfig{
		Base:         5,
		Variance:     2, // Not used for exponential
		Distribution: "exponential",
		Min:          1,
		Max:          30,
	}

	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples
	for i := 0; i < 100; i++ {
		duration := config.GetThinkTime(rng)
		seconds := duration.Seconds()

		// Should be within min/max bounds
		assert.GreaterOrEqual(t, seconds, float64(config.Min))
		assert.LessOrEqual(t, seconds, float64(config.Max))
	}
}

func TestUserConfig_GetUserCount(t *testing.T) {
	config := UserConfig{
		PerOrg: RangeConfig{Min: 10, Max: 50},
	}

	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples
	for i := 0; i < 100; i++ {
		count := config.GetUserCount(rng)
		assert.GreaterOrEqual(t, count, 10)
		assert.LessOrEqual(t, count, 50)
	}

	// Test exact value when min == max
	exactConfig := UserConfig{
		PerOrg: RangeConfig{Min: 25, Max: 25},
	}
	count := exactConfig.GetUserCount(rng)
	assert.Equal(t, 25, count)
}

func TestRangeConfig_GetValue(t *testing.T) {
	config := RangeConfig{Min: 5, Max: 15}
	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples
	for i := 0; i < 100; i++ {
		value := config.GetValue(rng)
		assert.GreaterOrEqual(t, value, 5)
		assert.LessOrEqual(t, value, 15)
	}
}

func TestDurationConfig_GetDuration(t *testing.T) {
	config := DurationConfig{Min: 60, Max: 120}
	rng := NewDefaultRNG(rand.New(rand.NewSource(42)))

	// Test multiple samples
	for i := 0; i < 100; i++ {
		duration := config.GetDuration(rng)
		assert.GreaterOrEqual(t, duration, 60*time.Second)
		assert.LessOrEqual(t, duration, 120*time.Second)
	}
}
