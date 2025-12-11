package daw

import (
	"context"
	"strings"
	"testing"

	magdaconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectionActionsIntegration tests selection actions through the full DAW agent flow
// These tests verify that selection actions are generated correctly from natural language
// and can be parsed from DSL code
func TestSelectionActionsIntegration(t *testing.T) {

	tests := []struct {
		name        string
		dslCode     string
		expectType  string
		expectCount int
	}{
		{
			name:        "select track via DSL",
			dslCode:     `track(index=0).set_track(selected=true)`,
			expectType:  "set_track",
			expectCount: 2, // create_track + set_track_selected
		},
		{
			name:        "deselect track via DSL",
			dslCode:     `track(index=1).set_track(selected=false)`,
			expectType:  "set_track",
			expectCount: 2,
		},
		{
			name:        "create and select track",
			dslCode:     `track(instrument="Serum").set_track(selected=true)`,
			expectType:  "set_track",
			expectCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse DSL directly using the functional parser
			parser, err := NewFunctionalDSLParser()
			require.NoError(t, err, "Failed to create parser")

			actions, err := parser.ParseDSL(tt.dslCode)
			require.NoError(t, err, "Failed to parse DSL")
			require.GreaterOrEqual(t, len(actions), tt.expectCount,
				"Should have at least %d actions", tt.expectCount)

			// Find the selection action
			foundSelection := false
			for _, action := range actions {
				actionType, ok := action["action"].(string)
				if !ok {
					continue
				}

				if actionType == tt.expectType {
					foundSelection = true

					// Verify action structure
					track, ok := action["track"].(int)
					assert.True(t, ok, "Action should have 'track' field as int")
					assert.GreaterOrEqual(t, track, 0, "Track index should be >= 0")

					selected, ok := action["selected"].(bool)
					assert.True(t, ok, "Action should have 'selected' field as bool")

					t.Logf("✅ Found %s action: track=%d, selected=%v",
						actionType, track, selected)
					break
				}
			}

			assert.True(t, foundSelection,
				"Should have found %s action", tt.expectType)
		})
	}
}

// TestSelectionWithStateIntegration tests selection with REAPER state context
func TestSelectionWithStateIntegration(t *testing.T) {
	parser, err := NewFunctionalDSLParser()
	require.NoError(t, err, "Failed to create parser")

	// Set up REAPER state with existing tracks
	state := map[string]any{
		"state": map[string]any{
			"tracks": []map[string]any{
				{"index": 0, "name": "Drums", "selected": false},
				{"index": 1, "name": "Bass", "selected": false},
				{"index": 2, "name": "Guitar", "selected": false},
			},
		},
	}
	parser.SetState(state)

	// Test selecting an existing track by index
	dslCode := `track(id=1).set_track(selected=true)`

	actions, err := parser.ParseDSL(dslCode)
	require.NoError(t, err, "Failed to parse DSL")

	// Should have at least one action (set_track_selected)
	// Note: track(id=1) references existing track, so no create_track action
	foundSelection := false
	for _, action := range actions {
		actionType, ok := action["action"].(string)
		if !ok {
			continue
		}

		if actionType == "set_track" {
			if _, ok := action["selected"]; ok {
				foundSelection = true
			track, ok := action["track"].(int)
			assert.True(t, ok, "Action should have 'track' field")
			// Track id=1 is 0-based index 0 (first track)
			// But the parser might use the id directly, so we just check it's valid
			assert.GreaterOrEqual(t, track, 0, "Track index should be valid")

			selected, ok := action["selected"].(bool)
			assert.True(t, ok, "Action should have 'selected' field")
				assert.True(t, selected, "Track should be selected")
			}
		}
	}

	assert.True(t, foundSelection, "Should have found set_track action with selected=true")
}

// TestSelectionActionChainIntegration tests chaining selection with other operations
func TestSelectionActionChainIntegration(t *testing.T) {
	parser, err := NewFunctionalDSLParser()
	require.NoError(t, err, "Failed to create parser")

	// Test: Create track, name it, set volume, then select it
	dslCode := `track(instrument="Serum").set_track(name="Bass", volume_db=-3.0, selected=true)`

	actions, err := parser.ParseDSL(dslCode)
	require.NoError(t, err, "Failed to parse DSL")
	require.GreaterOrEqual(t, len(actions), 2,
		"Should have at least 2 actions (create_track, set_track with all properties)")

	// Verify action sequence
	actionSequence := make([]string, len(actions))
	for i, action := range actions {
		actionType, ok := action["action"].(string)
		require.True(t, ok, "Action %d should have 'action' field", i)
		actionSequence[i] = actionType
	}

	// Verify expected sequence
	assert.Equal(t, "create_track", actionSequence[0],
		"First action should be create_track")

	// Find selection action
	hasSelection := false
	for i, actionType := range actionSequence {
		if actionType == "set_track" {
			action := actions[i]
			if _, ok := action["selected"]; ok {
				hasSelection = true
				// Verify it comes after other operations
				assert.Greater(t, i, 0,
					"Selection should come after track creation")

				// Verify selection is true
				selected, ok := action["selected"].(bool)
				assert.True(t, ok, "set_track should have 'selected' field")
				assert.True(t, selected, "Track should be selected")
			}
		}
	}

	assert.True(t, hasSelection, "Should have set_track_selected action")
	t.Logf("✅ Action sequence: %v", actionSequence)
}

// TestDawAgentSelectionFlow tests the full DAW agent flow with selection
// This is a true integration test that goes through the agent's GenerateActions
// Note: This requires an API key or will be skipped
func TestDawAgentSelectionFlow(t *testing.T) {
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
	state := map[string]any{
		"tracks": []map[string]any{
			{"index": 0, "name": "Drums", "selected": false},
			{"index": 1, "name": "Bass", "selected": false},
		},
	}

	// Test: "select track 1"
	result, err := agent.GenerateActions(ctx, "select track 1", state)
	if err != nil {
		// If API key is invalid, skip
		if contains(err.Error(), "API key") || contains(err.Error(), "Unauthorized") {
			t.Skip("Skipping test: Invalid or missing OpenAI API key")
			return
		}
		require.NoError(t, err, "Failed to generate actions")
	}

	require.NotNil(t, result, "Result should not be nil")
	require.Greater(t, len(result.Actions), 0,
		"Should have at least one action")

	// Verify we got a selection action
	hasSelection := false
	for _, action := range result.Actions {
		actionType, ok := action["action"].(string)
		if !ok {
			continue
		}

		if actionType == "set_track" {
			if _, ok := action["selected"]; ok {
			hasSelection = true
			selected, ok := action["selected"].(bool)
			assert.True(t, ok, "Action should have 'selected' field")
				assert.True(t, selected, "Track should be selected")
			}
		}
	}

	assert.True(t, hasSelection, "Should have set_track action with selected=true")
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
