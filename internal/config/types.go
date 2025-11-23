package config

import "time"

// LoadTestConfig is the root configuration structure for a load test scenario.
// It follows the Kubernetes-like resource model with apiVersion, kind, metadata, and spec.
type LoadTestConfig struct {
	APIVersion string              `yaml:"apiVersion"` // e.g., "loadtest.ai-aas.dev/v1"
	Kind       string              `yaml:"kind"`       // e.g., "LoadTestScenario"
	Metadata   LoadTestMetadata    `yaml:"metadata"`
	Spec       LoadTestSpec        `yaml:"spec"`
}

// LoadTestMetadata contains identifying information for the test configuration.
type LoadTestMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// LoadTestSpec defines the complete test scenario parameters.
type LoadTestSpec struct {
	Organizations OrganizationConfig `yaml:"organizations"`
	Users         UserConfig         `yaml:"users"`
	UserBehavior  BehaviorConfig     `yaml:"userBehavior"`
	Targets       TargetConfig       `yaml:"targets"`
	Limits        LimitsConfig       `yaml:"limits,omitempty"`
	Scenarios     ScenarioMixConfig  `yaml:"scenarioMix,omitempty"`
	TestTypes     []TestTypeConfig   `yaml:"testTypes,omitempty"`
}

// OrganizationConfig defines how many organizations to create and their properties.
type OrganizationConfig struct {
	Count          int               `yaml:"count"`                    // Number of organizations to create
	NamePrefix     string            `yaml:"namePrefix,omitempty"`     // e.g., "loadtest-org"
	Budget         BudgetConfig      `yaml:"budget,omitempty"`         // Budget per org
	CustomMetadata map[string]string `yaml:"customMetadata,omitempty"` // Additional metadata
}

// BudgetConfig defines budget limits for organizations.
type BudgetConfig struct {
	LimitUSD     float64 `yaml:"limitUSD,omitempty"`     // Total budget limit
	DailyUSD     float64 `yaml:"dailyUSD,omitempty"`     // Daily budget limit
	WarnAtUSD    float64 `yaml:"warnAtUSD,omitempty"`    // Warning threshold
	EnableAlerts bool    `yaml:"enableAlerts,omitempty"` // Enable budget alerts
}

// UserConfig defines how many users to create per organization.
type UserConfig struct {
	PerOrg     RangeConfig       `yaml:"perOrg"`               // Number of users per org (min/max)
	NamePrefix string            `yaml:"namePrefix,omitempty"` // e.g., "loadtest-user"
	APIKeys    APIKeyConfig      `yaml:"apiKeys,omitempty"`    // API key configuration
	Roles      []string          `yaml:"roles,omitempty"`      // User roles to assign
	Metadata   map[string]string `yaml:"metadata,omitempty"`   // User metadata
}

// RangeConfig defines a numeric range with min and max values.
type RangeConfig struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

// APIKeyConfig defines API key creation parameters.
type APIKeyConfig struct {
	PerUser    int    `yaml:"perUser,omitempty"`    // Number of API keys per user
	NamePrefix string `yaml:"namePrefix,omitempty"` // e.g., "loadtest-key"
	ExpiryDays int    `yaml:"expiryDays,omitempty"` // Days until expiry (0 = no expiry)
}

// BehaviorConfig defines how users should behave during the test.
type BehaviorConfig struct {
	ThinkTimeSeconds    ThinkTimeConfig       `yaml:"thinkTimeSeconds"`
	SessionDuration     DurationConfig        `yaml:"sessionDuration,omitempty"`
	QuestionsPerSession RangeConfig           `yaml:"questionsPerSession,omitempty"`
	ConversationStyle   ConversationConfig    `yaml:"conversationStyle,omitempty"`
	ErrorHandling       ErrorHandlingConfig   `yaml:"errorHandling,omitempty"`
}

// ThinkTimeConfig defines the wait time between requests with variance.
// Implements the "base ± variance" pattern requested by the user.
type ThinkTimeConfig struct {
	Base         int    `yaml:"base"`                   // Base think time in seconds (e.g., 5)
	Variance     int    `yaml:"variance"`               // Variance in seconds (e.g., 2 means ±2)
	Distribution string `yaml:"distribution,omitempty"` // "uniform" (default), "gaussian", "exponential"
	Min          int    `yaml:"min,omitempty"`          // Minimum allowed think time
	Max          int    `yaml:"max,omitempty"`          // Maximum allowed think time
}

// DurationConfig defines a time duration range.
type DurationConfig struct {
	Min int    `yaml:"min"` // Minimum duration in seconds
	Max int    `yaml:"max"` // Maximum duration in seconds
	Unit string `yaml:"unit,omitempty"` // "seconds", "minutes", "hours"
}

// ConversationConfig defines multi-turn conversation parameters.
type ConversationConfig struct {
	MultiTurnProbability float64     `yaml:"multiTurnProbability,omitempty"` // 0.0-1.0 probability of multi-turn
	TurnsPerConversation RangeConfig `yaml:"turnsPerConversation,omitempty"` // Number of turns
	ContextRetention     bool        `yaml:"contextRetention,omitempty"`     // Keep conversation context
}

// ErrorHandlingConfig defines how to handle errors during testing.
type ErrorHandlingConfig struct {
	RetryAttempts    int      `yaml:"retryAttempts,omitempty"`    // Number of retries on error
	RetryBackoffMs   int      `yaml:"retryBackoffMs,omitempty"`   // Backoff between retries
	IgnoreErrorCodes []int    `yaml:"ignoreErrorCodes,omitempty"` // HTTP codes to ignore
	StopOnErrors     int      `yaml:"stopOnErrors,omitempty"`     // Stop after N errors
}

// TargetConfig defines the API endpoints to test.
type TargetConfig struct {
	APIRouterURL string            `yaml:"apiRouterUrl"`           // Main API endpoint (e.g., https://api.platform.com)
	UserOrgURL   string            `yaml:"userOrgUrl"`             // User/Org service endpoint for bootstrap
	AdminAPIKey  string            `yaml:"adminApiKey,omitempty"`  // Admin API key for bootstrap (optional)
	TLSVerify    bool              `yaml:"tlsVerify,omitempty"`    // Verify TLS certificates
	Timeout      int               `yaml:"timeout,omitempty"`      // Request timeout in seconds
	Headers      map[string]string `yaml:"headers,omitempty"`      // Additional headers
}

// LimitsConfig defines test execution limits.
type LimitsConfig struct {
	MaxCostUSD        float64 `yaml:"maxCostUSD,omitempty"`        // Stop test if total cost exceeds
	MaxRequests       int     `yaml:"maxRequests,omitempty"`       // Stop test after N requests
	MaxDurationSec    int     `yaml:"maxDurationSec,omitempty"`    // Stop test after N seconds
	MaxErrorRate      float64 `yaml:"maxErrorRate,omitempty"`      // Stop if error rate exceeds (0.0-1.0)
	MaxConcurrentReqs int     `yaml:"maxConcurrentReqs,omitempty"` // Max concurrent requests
}

// ScenarioMixConfig defines the mix of different test scenarios.
// Implements the template-based scenario framework requested by the user.
type ScenarioMixConfig struct {
	Scenarios []ScenarioWeight `yaml:"scenarios"`
}

// ScenarioWeight defines a scenario and its weight in the mix.
type ScenarioWeight struct {
	Name   string `yaml:"name"`   // Scenario name (references TestScenario resource)
	Weight int    `yaml:"weight"` // Weight (e.g., 75 for 75%)
}

// TestTypeConfig defines different test types (random_short, long, cached).
type TestTypeConfig struct {
	Name              string               `yaml:"name"`                        // e.g., "random_short", "long", "cached"
	Weight            int                  `yaml:"weight"`                      // Percentage of requests using this type
	QuestionStrategy  string               `yaml:"questionStrategy,omitempty"`  // "historical", "mathematical", "mixed", etc.
	CacheBehavior     string               `yaml:"cacheBehavior,omitempty"`     // "avoid", "prefer", "mixed"
	CacheSalt         string               `yaml:"cacheSalt,omitempty"`         // Cache salt for isolation testing
	ModelTargeting    ModelTargetingConfig `yaml:"modelTargeting,omitempty"`    // Model selection rules
	DocumentLibrary   DocumentConfig       `yaml:"documentLibrary,omitempty"`   // Document-based query config
	ContinuousLoop    bool                 `yaml:"continuousLoop,omitempty"`    // Restart with new random seed
	StreamingResponse bool                 `yaml:"streamingResponse,omitempty"` // Use streaming API
}

// ModelTargetingConfig defines rules for model selection.
type ModelTargetingConfig struct {
	SLMModels    []string           `yaml:"slmModels,omitempty"`    // Small language models
	MediumModels []string           `yaml:"mediumModels,omitempty"` // Medium language models
	Rules        []ModelTargetRule  `yaml:"rules,omitempty"`        // Routing rules
}

// ModelTargetRule defines a rule for routing to specific models.
type ModelTargetRule struct {
	Condition  string `yaml:"condition"`  // e.g., "query_length < 100"
	TargetType string `yaml:"targetType"` // "slm" or "medium"
}

// DocumentConfig defines document library integration.
type DocumentConfig struct {
	Enabled       bool     `yaml:"enabled,omitempty"`
	BucketName    string   `yaml:"bucketName,omitempty"`    // S3/MinIO bucket
	DocumentPaths []string `yaml:"documentPaths,omitempty"` // Paths to documents
	SampleSize    int      `yaml:"sampleSize,omitempty"`    // Number of documents to sample
}

// RuntimeConfig contains runtime state and credentials (not from YAML).
type RuntimeConfig struct {
	TestRunID     string                  // Unique ID for this test run
	WorkerID      string                  // Unique ID for this worker
	StartTime     time.Time               // Test start time
	Organizations []BootstrappedOrg       // Created organizations
	Users         []BootstrappedUser      // Created users
	APIKeys       map[string]string       // user_id -> api_key mapping
}

// BootstrappedOrg represents a created organization.
type BootstrappedOrg struct {
	ID          string
	Name        string
	BudgetID    string
	CreatedAt   time.Time
}

// BootstrappedUser represents a created user.
type BootstrappedUser struct {
	ID           string
	Name         string
	Email        string
	OrgID        string
	OrgName      string
	APIKeys      []BootstrappedAPIKey
	CreatedAt    time.Time
}

// BootstrappedAPIKey represents a created API key.
type BootstrappedAPIKey struct {
	ID          string
	Name        string
	Key         string // The actual API key value
	UserID      string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}
