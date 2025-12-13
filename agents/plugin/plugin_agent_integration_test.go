package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginProcessingWithLargeList tests the plugin processing endpoint
// with a large list of plugins (e.g., 737 plugins from REAPER)
func TestPluginProcessingWithLargeList(t *testing.T) {
	// Load environment variables
	godotenv.Load()

	// Get API URL from environment or use default
	apiURL := os.Getenv("AIDEAS_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	// Get auth token
	email := os.Getenv("AIDEAS_EMAIL")
	password := os.Getenv("AIDEAS_PASSWORD")
	if email == "" || password == "" {
		t.Skip("Skipping integration test: AIDEAS_EMAIL and AIDEAS_PASSWORD not set")
	}

	// Authenticate first
	token, err := authenticate(apiURL, email, password)
	require.NoError(t, err, "Failed to authenticate")

	// Load plugins from test file
	plugins, err := loadPluginsFromTestFile()
	if err != nil {
		t.Skipf("Skipping test: Could not load plugins from test file: %v", err)
	}

	if len(plugins) == 0 {
		t.Skip("Skipping test: No plugins in test file")
	}

	t.Logf("Loaded %d plugins from test file", len(plugins))

	// Build request
	requestBody := map[string]interface{}{
		"plugins": plugins,
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	t.Logf("Request body size: %d bytes", len(jsonBody))

	// Make request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL+"/api/v1/magda/plugins/process",
		io.NopCloser(bytes.NewReader(jsonBody)))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	startTime := time.Now()
	client := &http.Client{
		Timeout: 180 * time.Second,
	}
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	require.NoError(t, err, "Request failed")
	defer resp.Body.Close()

	t.Logf("Request completed in %v", duration)

	// Read response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"Expected 200 OK, got %d: %s", resp.StatusCode, string(body))

	// Parse response
	var response struct {
		Plugins           []PluginInfo      `json:"plugins"`
		Aliases           map[string]string `json:"aliases"`
		OriginalCount     int               `json:"original_count"`
		DeduplicatedCount int               `json:"deduplicated_count"`
		AliasesCount      int               `json:"aliases_count"`
	}

	err = json.Unmarshal(body, &response)
	require.NoError(t, err, "Failed to parse response: %s", string(body))

	// Verify response structure
	assert.Greater(t, len(response.Plugins), 0, "Should have deduplicated plugins")
	assert.Greater(t, len(response.Aliases), 0, "Should have generated aliases")
	assert.Equal(t, len(plugins), response.OriginalCount, "Original count should match")
	assert.LessOrEqual(t, response.DeduplicatedCount, response.OriginalCount,
		"Deduplicated count should be <= original")
	assert.Equal(t, len(response.Aliases), response.AliasesCount, "Aliases count should match")

	t.Logf("✅ Test passed:")
	t.Logf("   Original plugins: %d", response.OriginalCount)
	t.Logf("   Deduplicated: %d", response.DeduplicatedCount)
	t.Logf("   Aliases generated: %d", response.AliasesCount)
	t.Logf("   Request duration: %v", duration)
}

// authenticate performs login and returns JWT token
func authenticate(apiURL, email, password string) (string, error) {
	requestBody := map[string]string{
		"email":    email,
		"password": password,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(apiURL+"/api/auth/login", "application/json",
		bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// loadPluginsFromTestFile loads plugins from the test file exported by REAPER extension
func loadPluginsFromTestFile() ([]PluginInfo, error) {
	// Try multiple possible locations
	possiblePaths := []string{
		"../../magda-reaper/test_data/plugins.json",
		"../magda-reaper/test_data/plugins.json",
		"test_data/plugins.json",
		"/tmp/magda_test_plugins.json",
	}

	var filePath string
	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			filePath = absPath
			break
		}
	}

	if filePath == "" {
		return nil, fmt.Errorf("plugins test file not found in any of: %v", possiblePaths)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read test file: %w", err)
	}

	var result struct {
		Plugins []PluginInfo `json:"plugins"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse test file: %w", err)
	}

	return result.Plugins, nil
}
