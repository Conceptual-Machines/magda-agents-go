package daw

import (
	"context"
	"log"
	"strings"
	"testing"

	magdaconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/stretchr/testify/require"
)

// TestFilterDSLGeneration tests what DSL code is generated for "select all tracks named foo"
// This is an isolated test to debug DSL generation issues
func TestFilterDSLGeneration(t *testing.T) {
	cfg := &magdaconfig.Config{
		OpenAIAPIKey: "", // Will skip if not set
	}

	agent := NewDawAgent(cfg)

	// If no API key, skip this test
	if cfg.OpenAIAPIKey == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set (integration test requires API key)")
		return
	}

	ctx := context.Background()
	state := map[string]interface{}{
		"tracks": []map[string]interface{}{
			{"index": 0, "name": "foo", "selected": false},
			{"index": 1, "name": "bar", "selected": false},
			{"index": 2, "name": "foo", "selected": false},
			{"index": 3, "name": "foo", "selected": false},
		},
	}

	// Test: "select all tracks named foo"
	log.Printf("🧪 Testing DSL generation for: 'select all tracks named foo'")
	result, err := agent.GenerateActions(ctx, "select all tracks named foo", state)
	if err != nil {
		// If API key is invalid, skip
		if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "Unauthorized") {
			t.Skip("Skipping test: Invalid or missing OpenAI API key")
			return
		}
		require.NoError(t, err, "Failed to generate actions")
	}

	require.NotNil(t, result, "Result should not be nil")
	log.Printf("✅ Generated %d actions", len(result.Actions))

	// Log all actions for debugging
	for i, action := range result.Actions {
		log.Printf("   Action %d: %+v", i+1, action)
	}

	// Verify we got selection actions
	selectionCount := 0
	for _, action := range result.Actions {
		actionType, ok := action["action"].(string)
		if !ok {
			continue
		}

		if actionType == "set_track_selected" {
			selectionCount++
			track, ok := action["track"].(int)
			if !ok {
				if trackFloat, ok := action["track"].(float64); ok {
					track = int(trackFloat)
				}
			}
			selected, ok := action["selected"].(bool)
			require.True(t, ok, "Action should have 'selected' field")
			log.Printf("   ✅ Selection action: track=%d, selected=%v", track, selected)
		}
	}

	log.Printf("📊 Total selection actions: %d (expected: 3)", selectionCount)
	require.Equal(t, 3, selectionCount, "Should have 3 selection actions for 3 'foo' tracks")
}
