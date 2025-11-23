package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/otherjamesbrown/ai-aas-loadtest/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBootstrap_CreateOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/organizations", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "test-org", payload["name"])

		response := map[string]interface{}{
			"id":         "org-123",
			"name":       "test-org",
			"budget_id":  "budget-123",
			"created_at": time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "", 10*time.Second, logger)

	budget := config.BudgetConfig{
		LimitUSD: 100.0,
		DailyUSD: 50.0,
	}

	org, err := manager.createOrganization("test-org", &budget)
	require.NoError(t, err)
	assert.Equal(t, "org-123", org.ID)
	assert.Equal(t, "test-org", org.Name)
	assert.Equal(t, "budget-123", org.BudgetID)
}

func TestBootstrap_CreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/users", r.URL.Path)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "test-user", payload["name"])
		assert.Equal(t, "test-user@loadtest.example.com", payload["email"])
		assert.Equal(t, "org-123", payload["organization_id"])

		response := map[string]interface{}{
			"id":                "user-456",
			"name":              "test-user",
			"email":             "test-user@loadtest.example.com",
			"organization_id":   "org-123",
			"organization_name": "test-org",
			"created_at":        time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "", 10*time.Second, logger)

	user, err := manager.createUser("org-123", "test-user")
	require.NoError(t, err)
	assert.Equal(t, "user-456", user.ID)
	assert.Equal(t, "test-user", user.Name)
	assert.Equal(t, "test-user@loadtest.example.com", user.Email)
	assert.Equal(t, "org-123", user.OrgID)
}

func TestBootstrap_CreateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/api-keys", r.URL.Path)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "test-key", payload["name"])
		assert.Equal(t, "user-456", payload["user_id"])

		response := map[string]interface{}{
			"id":         "key-789",
			"name":       "test-key",
			"key":        "sk-test-abc123xyz",
			"user_id":    "user-456",
			"created_at": time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "", 10*time.Second, logger)

	apiKey, err := manager.createAPIKey("user-456", "test-key", 7)
	require.NoError(t, err)
	assert.Equal(t, "key-789", apiKey.ID)
	assert.Equal(t, "test-key", apiKey.Name)
	assert.Equal(t, "sk-test-abc123xyz", apiKey.Key)
	assert.Equal(t, "user-456", apiKey.UserID)
}

func TestBootstrap_FullBootstrap(t *testing.T) {
	orgCount := 0
	userCount := 0
	keyCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/organizations":
			orgCount++
			response := map[string]interface{}{
				"id":         "org-" + string(rune(orgCount)),
				"name":       "test-org",
				"created_at": time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(response)

		case "/api/v1/users":
			userCount++
			response := map[string]interface{}{
				"id":              "user-" + string(rune(userCount)),
				"name":            "test-user",
				"email":           "test@example.com",
				"organization_id": "org-1",
				"created_at":      time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(response)

		case "/api/v1/api-keys":
			keyCount++
			response := map[string]interface{}{
				"id":         "key-" + string(rune(keyCount)),
				"name":       "test-key",
				"key":        "sk-test-key-" + string(rune(keyCount)),
				"user_id":    "user-1",
				"created_at": time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "", 10*time.Second, logger)

	cfg := &config.LoadTestConfig{
		Spec: config.LoadTestSpec{
			Organizations: config.OrganizationConfig{
				Count:      2,
				NamePrefix: "test-org",
			},
			Users: config.UserConfig{
				PerOrg: config.RangeConfig{
					Min: 3,
					Max: 3,
				},
				NamePrefix: "test-user",
				APIKeys: config.APIKeyConfig{
					PerUser:    1,
					NamePrefix: "test-key",
				},
			},
		},
	}

	runtime := config.NewRuntimeConfig()
	err := manager.Bootstrap(cfg, runtime)
	require.NoError(t, err)

	assert.Equal(t, 2, len(runtime.Organizations))
	assert.Equal(t, 6, len(runtime.Users)) // 2 orgs * 3 users
	assert.Equal(t, 6, len(runtime.APIKeys)) // 6 users * 1 key each

	// Verify API call counts
	assert.Equal(t, 2, orgCount)
	assert.Equal(t, 6, userCount)
	assert.Equal(t, 6, keyCount)
}

func TestBootstrap_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "", 10*time.Second, logger)

	budget := config.BudgetConfig{}
	_, err := manager.createOrganization("test-org", &budget)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed with status 500")
}

func TestBootstrap_AdminKeyHeader(t *testing.T) {
	receivedAuth := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"id":         "org-123",
			"name":       "test-org",
			"created_at": time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	manager := NewManager(server.URL, "admin-key-abc", 10*time.Second, logger)

	budget := config.BudgetConfig{}
	_, err := manager.createOrganization("test-org", &budget)
	require.NoError(t, err)
	assert.Equal(t, "Bearer admin-key-abc", receivedAuth)
}
