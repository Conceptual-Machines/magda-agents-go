package daw

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-agents-go/llm"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file from project root for tests
	// Try multiple paths in case tests are run from different directories
	_ = godotenv.Load()                      // Current directory
	_ = godotenv.Load(".env")               // Current directory explicit
	_ = godotenv.Load("../.env")            // Parent directory
	_ = godotenv.Load("../../.env")         // Project root from agents/daw/
	
	// Also try to find project root by looking for go.mod
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// getTestConfig returns a test config, skipping if API key is not available
func getTestConfig(t *testing.T) *config.Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
	return &config.Config{
		OpenAIAPIKey: apiKey,
	}
}

func TestDawAgent_OutOfScope_Request(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	agent := NewDawAgent(cfg)
	ctx := context.Background()

	tests := []struct {
		name           string
		question       string
		description    string
		expectError    bool
		errorContains  string
	}{
		{
			name:          "cooking_request",
			question:      "bake me a cake",
			description:   "Cooking task - completely out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "email_request",
			question:      "send an email to john@example.com",
			description:   "Email task - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "weather_request",
			question:      "what's the weather today?",
			description:   "Weather query - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "general_question",
			question:      "what is 2+2?",
			description:   "General math question - out of scope",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "valid_daw_request",
			question:      "add reverb to track 1",
			description:   "Valid DAW operation - should succeed",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "valid_musical_request",
			question:      "create a chord progression I VI IV V",
			description:   "Valid musical content - should succeed",
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			result, err := agent.GenerateActions(ctx, tt.question, nil)

			if tt.expectError {
				require.Error(t, err, "Expected error for out-of-scope request")
				t.Logf("✅ Got expected error: %v", err)
				
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains),
						"Error message should contain '%s'", tt.errorContains)
				}
				
				// Verify error format
				assert.Contains(t, err.Error(), "out of scope",
					"Error should indicate request is out of scope")
				
				// Should not have actions
				if result != nil {
					assert.Empty(t, result.Actions, "Out-of-scope request should not produce actions")
				}
			} else {
				require.NoError(t, err, "Valid request should not error")
				require.NotNil(t, result, "Valid request should return result")
				t.Logf("✅ Request succeeded with %d actions", len(result.Actions))
			}
		})
	}
}

func TestDawAgent_ParseErrorComment(t *testing.T) {
	cfg := getTestConfigForKeywords(t)
	agent := NewDawAgent(cfg)

	tests := []struct {
		name          string
		rawOutput     string
		expectError   bool
		errorContains string
	}{
		{
			name:          "error_comment_format",
			rawOutput:     "// ERROR: This request is out of scope. MAGDA only handles music production and REAPER/DAW operations, not cooking tasks.",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "error_comment_with_whitespace",
			rawOutput:     "  // ERROR: Request cannot be handled  ",
			expectError:   true,
			errorContains: "out of scope",
		},
		{
			name:          "valid_dsl_code",
			rawOutput:     "track(instrument=\"Serum\").new_clip(bar=1, length_bars=4)",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "error_comment_multiline",
			rawOutput:     "// ERROR: This request is out of scope.\n// Additional explanation here.",
			expectError:   true,
			errorContains: "out of scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &llm.GenerationResponse{
				RawOutput: tt.rawOutput,
			}

			actions, err := agent.parseActionsFromResponse(resp, nil)

			if tt.expectError {
				require.Error(t, err, "Expected error for error comment format")
				t.Logf("✅ Got expected error: %v", err)
				
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains),
						"Error message should contain '%s'", tt.errorContains)
				}
				
				assert.Nil(t, actions, "Error comment should not produce actions")
			} else {
				require.NoError(t, err, "Valid DSL should not error")
				require.NotNil(t, actions, "Valid DSL should produce actions")
			}
		})
	}
}

// getTestConfigForKeywords returns a test config for keyword-only tests (no API key needed)
func getTestConfigForKeywords(t *testing.T) *config.Config {
	return &config.Config{
		OpenAIAPIKey: "test-key", // Not used for parsing tests
	}
}

