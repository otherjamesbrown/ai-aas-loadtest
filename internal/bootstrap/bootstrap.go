package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/otherjamesbrown/ai-aas-loadtest/internal/config"
	"go.uber.org/zap"
)

// Manager handles the bootstrapping of organizations, users, and API keys.
type Manager struct {
	userOrgURL string
	adminKey   string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewManager creates a new bootstrap manager.
func NewManager(userOrgURL, adminKey string, timeout time.Duration, logger *zap.Logger) *Manager {
	return &Manager{
		userOrgURL: userOrgURL,
		adminKey:   adminKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// Bootstrap creates all organizations, users, and API keys as specified in the config.
// It returns the populated RuntimeConfig with all created entities.
func (m *Manager) Bootstrap(cfg *config.LoadTestConfig, runtime *config.RuntimeConfig) error {
	m.logger.Info("Starting bootstrap process",
		zap.String("test_run_id", runtime.TestRunID),
		zap.Int("org_count", cfg.Spec.Organizations.Count),
	)

	// Create organizations
	for i := 0; i < cfg.Spec.Organizations.Count; i++ {
		orgName := fmt.Sprintf("%s-%d", cfg.Spec.Organizations.NamePrefix, i+1)
		if cfg.Spec.Organizations.NamePrefix == "" {
			orgName = fmt.Sprintf("loadtest-org-%s-%d", runtime.TestRunID, i+1)
		}

		org, err := m.createOrganization(orgName, &cfg.Spec.Organizations.Budget)
		if err != nil {
			return fmt.Errorf("failed to create organization %s: %w", orgName, err)
		}

		runtime.Organizations = append(runtime.Organizations, *org)
		m.logger.Info("Created organization",
			zap.String("org_id", org.ID),
			zap.String("org_name", org.Name),
		)

		// Create users for this organization
		userCount := cfg.Spec.Users.PerOrg.Min
		if cfg.Spec.Users.PerOrg.Max > cfg.Spec.Users.PerOrg.Min {
			// For bootstrap, use the minimum. Runtime will handle variance.
			userCount = cfg.Spec.Users.PerOrg.Min
		}

		for j := 0; j < userCount; j++ {
			userName := fmt.Sprintf("%s-%d", cfg.Spec.Users.NamePrefix, j+1)
			if cfg.Spec.Users.NamePrefix == "" {
				userName = fmt.Sprintf("loadtest-user-%d", j+1)
			}

			user, err := m.createUser(org.ID, userName)
			if err != nil {
				return fmt.Errorf("failed to create user %s in org %s: %w", userName, org.Name, err)
			}

			// Create API keys for this user
			keysPerUser := 1
			if cfg.Spec.Users.APIKeys.PerUser > 0 {
				keysPerUser = cfg.Spec.Users.APIKeys.PerUser
			}

			for k := 0; k < keysPerUser; k++ {
				keyName := fmt.Sprintf("%s-%d", cfg.Spec.Users.APIKeys.NamePrefix, k+1)
				if cfg.Spec.Users.APIKeys.NamePrefix == "" {
					keyName = fmt.Sprintf("loadtest-key-%d", k+1)
				}

				apiKey, err := m.createAPIKey(user.ID, keyName, cfg.Spec.Users.APIKeys.ExpiryDays)
				if err != nil {
					return fmt.Errorf("failed to create API key %s for user %s: %w", keyName, user.Name, err)
				}

				user.APIKeys = append(user.APIKeys, *apiKey)

				// Store in runtime for easy lookup
				runtime.APIKeys[user.ID] = apiKey.Key
			}

			runtime.Users = append(runtime.Users, *user)
			m.logger.Info("Created user",
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
				zap.String("org_name", org.Name),
				zap.Int("api_keys", len(user.APIKeys)),
			)
		}
	}

	m.logger.Info("Bootstrap complete",
		zap.Int("organizations", len(runtime.Organizations)),
		zap.Int("users", len(runtime.Users)),
		zap.Int("api_keys", len(runtime.APIKeys)),
	)

	return nil
}

// createOrganization creates a new organization via the User/Org Service API.
func (m *Manager) createOrganization(name string, budget *config.BudgetConfig) (*config.BootstrappedOrg, error) {
	payload := map[string]interface{}{
		"name": name,
	}

	// Add budget if specified
	if budget != nil && budget.LimitUSD > 0 {
		payload["budget"] = map[string]interface{}{
			"limit_usd":     budget.LimitUSD,
			"daily_usd":     budget.DailyUSD,
			"warn_at_usd":   budget.WarnAtUSD,
			"enable_alerts": budget.EnableAlerts,
		}
	}

	resp, err := m.makeRequest("POST", "/api/v1/organizations", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		BudgetID  string    `json:"budget_id,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse organization response: %w", err)
	}

	return &config.BootstrappedOrg{
		ID:        result.ID,
		Name:      result.Name,
		BudgetID:  result.BudgetID,
		CreatedAt: result.CreatedAt,
	}, nil
}

// createUser creates a new user via the User/Org Service API.
func (m *Manager) createUser(orgID, name string) (*config.BootstrappedUser, error) {
	email := fmt.Sprintf("%s@loadtest.example.com", name)

	payload := map[string]interface{}{
		"name":            name,
		"email":           email,
		"organization_id": orgID,
	}

	resp, err := m.makeRequest("POST", "/api/v1/users", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		OrgID     string    `json:"organization_id"`
		OrgName   string    `json:"organization_name,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	return &config.BootstrappedUser{
		ID:        result.ID,
		Name:      result.Name,
		Email:     result.Email,
		OrgID:     result.OrgID,
		OrgName:   result.OrgName,
		APIKeys:   []config.BootstrappedAPIKey{},
		CreatedAt: result.CreatedAt,
	}, nil
}

// createAPIKey creates a new API key via the User/Org Service API.
func (m *Manager) createAPIKey(userID, name string, expiryDays int) (*config.BootstrappedAPIKey, error) {
	payload := map[string]interface{}{
		"name":    name,
		"user_id": userID,
	}

	if expiryDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, expiryDays)
		payload["expires_at"] = expiresAt.Format(time.RFC3339)
	}

	resp, err := m.makeRequest("POST", "/api/v1/api-keys", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Key       string     `json:"key"` // The actual API key value
		UserID    string     `json:"user_id"`
		CreatedAt time.Time  `json:"created_at"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse API key response: %w", err)
	}

	return &config.BootstrappedAPIKey{
		ID:        result.ID,
		Name:      result.Name,
		Key:       result.Key,
		UserID:    result.UserID,
		CreatedAt: result.CreatedAt,
		ExpiresAt: result.ExpiresAt,
	}, nil
}

// makeRequest makes an HTTP request to the User/Org Service API.
func (m *Manager) makeRequest(method, path string, payload interface{}) ([]byte, error) {
	url := m.userOrgURL + path

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.adminKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.adminKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Cleanup deletes all created organizations, users, and API keys.
// This is useful for cleaning up after tests complete.
func (m *Manager) Cleanup(runtime *config.RuntimeConfig) error {
	m.logger.Info("Starting cleanup process",
		zap.String("test_run_id", runtime.TestRunID),
	)

	// Delete API keys
	for _, user := range runtime.Users {
		for _, key := range user.APIKeys {
			if err := m.deleteAPIKey(key.ID); err != nil {
				m.logger.Error("Failed to delete API key",
					zap.String("key_id", key.ID),
					zap.Error(err),
				)
			}
		}
	}

	// Delete users
	for _, user := range runtime.Users {
		if err := m.deleteUser(user.ID); err != nil {
			m.logger.Error("Failed to delete user",
				zap.String("user_id", user.ID),
				zap.Error(err),
			)
		}
	}

	// Delete organizations
	for _, org := range runtime.Organizations {
		if err := m.deleteOrganization(org.ID); err != nil {
			m.logger.Error("Failed to delete organization",
				zap.String("org_id", org.ID),
				zap.Error(err),
			)
		}
	}

	m.logger.Info("Cleanup complete")
	return nil
}

func (m *Manager) deleteAPIKey(keyID string) error {
	_, err := m.makeRequest("DELETE", "/api/v1/api-keys/"+keyID, nil)
	return err
}

func (m *Manager) deleteUser(userID string) error {
	_, err := m.makeRequest("DELETE", "/api/v1/users/"+userID, nil)
	return err
}

func (m *Manager) deleteOrganization(orgID string) error {
	_, err := m.makeRequest("DELETE", "/api/v1/organizations/"+orgID, nil)
	return err
}
