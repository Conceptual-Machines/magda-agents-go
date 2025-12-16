package coordination

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrator_Integration_DrummerAndDAW(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		question    string
		description string
		validate    func(t *testing.T, result *OrchestratorResult)
	}{
		{
			name:        "drum_track_with_breakbeat",
			question:    "create a drum track with Addictive Drums and add a breakbeat with ghost snares",
			description: "Creates drum track with instrument and breakbeat pattern with ghost snares",
			validate: func(t *testing.T, result *OrchestratorResult) {
				t.Helper()
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				// Should have track creation action from DAW agent
				hasTrackCreation := false
				hasDrumPattern := false

				for _, action := range result.Actions {
					actionType, _ := action["action"].(string)
					drumType, _ := action["type"].(string)

					if drumType == "drum_pattern" {
						hasDrumPattern = true
						t.Logf("🥁 Found drum_pattern: drum=%v, grid=%v", action["drum"], action["grid"])
					}

					if actionType == "create_track" {
						hasTrackCreation = true
						if instrument, ok := action["instrument"].(string); ok {
							t.Logf("🎹 Track instrument: %s", instrument)
						}
					}
				}

				assert.True(t, hasTrackCreation, "Should have track creation action from DAW agent")
				assert.True(t, hasDrumPattern, "Should have drum pattern action from Drummer agent")
			},
		},
		{
			name:        "kick_pattern_four_on_floor",
			question:    "create a drum track and add a four on the floor kick pattern",
			description: "Creates track and adds basic kick pattern",
			validate: func(t *testing.T, result *OrchestratorResult) {
				t.Helper()
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				hasTrackCreation := false
				hasDrumPattern := false

				for _, action := range result.Actions {
					actionType, _ := action["action"].(string)
					drumType, _ := action["type"].(string)

					if actionType == "create_track" {
						hasTrackCreation = true
					}

					if drumType == "drum_pattern" {
						hasDrumPattern = true
						t.Logf("🥁 Found drum_pattern: drum=%v", action["drum"])
					}
				}

				assert.True(t, hasTrackCreation, "Should have track creation action")
				assert.True(t, hasDrumPattern, "Should have drum_pattern action")
			},
		},
		{
			name:        "full_drum_kit_pattern",
			question:    "create a track with drums and add a rock beat with kick, snare on 2 and 4, and hi-hat eighth notes",
			description: "Creates complete drum pattern with multiple elements",
			validate: func(t *testing.T, result *OrchestratorResult) {
				t.Helper()
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				hasTrackCreation := false
				hasDrumPattern := false

				for _, action := range result.Actions {
					actionType, _ := action["action"].(string)
					drumType, _ := action["type"].(string)

					if actionType == "create_track" {
						hasTrackCreation = true
					}

					if drumType == "drum_pattern" {
						hasDrumPattern = true
						t.Logf("🥁 Found drum_pattern: drum=%v", action["drum"])
					}
				}

				assert.True(t, hasTrackCreation, "Should have track creation action")
				assert.True(t, hasDrumPattern, "Should have drum_pattern action")
			},
		},
		{
			name:        "drummer_only_simple_beat",
			question:    "add a simple drum beat",
			description: "Drummer agent only - no explicit track creation",
			validate: func(t *testing.T, result *OrchestratorResult) {
				t.Helper()
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				hasDrumPattern := false
				for _, action := range result.Actions {
					drumType, _ := action["type"].(string)

					if drumType == "drum_pattern" {
						hasDrumPattern = true
						t.Logf("🥁 Found drum_pattern: drum=%v", action["drum"])
					}
				}

				assert.True(t, hasDrumPattern, "Should have drum_pattern action")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			t.Logf("Test: %s", tt.description)
			t.Logf("Question: %q", tt.question)
			t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			start := time.Now()

			// Execute orchestrator
			result, err := orchestrator.GenerateActions(ctx, tt.question, nil)

			duration := time.Since(start)
			t.Logf("⏱️  Execution time: %v", duration)

			if err != nil {
				t.Logf("❌ Error: %v", err)
				if ctx.Err() == context.DeadlineExceeded {
					t.Fatal("Test timed out")
				}
				t.Fatalf("Orchestrator error: %v", err)
			}

			// Log result structure
			if result != nil {
				t.Logf("✅ Result received: %d actions", len(result.Actions))

				// Pretty print actions for debugging
				actionsJSON, _ := json.MarshalIndent(result.Actions, "", "  ")
				t.Logf("📋 Actions:\n%s", string(actionsJSON))

				// Validate result
				tt.validate(t, result)
			} else {
				t.Fatal("Result is nil")
			}

			t.Logf("✅ Test completed successfully")
		})
	}
}

func TestOrchestrator_Integration_DrummerKeywordDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfigForKeywords(t)
	orchestrator := NewOrchestrator(cfg)

	tests := []struct {
		name            string
		question        string
		expectedDAW     bool
		expectedDrummer bool
		description     string
	}{
		{
			name:            "drum_beat",
			question:        "create a drum beat",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Basic drum beat request",
		},
		{
			name:            "kick_pattern",
			question:        "add a kick pattern",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Kick drum pattern",
		},
		{
			name:            "hi_hat_groove",
			question:        "make a hi-hat groove",
			expectedDAW:     false,
			expectedDrummer: true,
			description:     "Hi-hat pattern",
		},
		{
			name:            "four_on_floor",
			question:        "four on the floor rhythm",
			expectedDAW:     false,
			expectedDrummer: true,
			description:     "Classic dance rhythm",
		},
		{
			name:            "breakbeat",
			question:        "add a breakbeat pattern",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Breakbeat pattern",
		},
		{
			name:            "snare_backbeat",
			question:        "snare on 2 and 4",
			expectedDAW:     false,
			expectedDrummer: true,
			description:     "Snare backbeat",
		},
		{
			name:            "percussion",
			question:        "add percussion to the track",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "General percussion",
		},
		{
			name:            "drum_fill",
			question:        "add a drum fill",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Drum fill",
		},
		{
			name:            "tom_pattern",
			question:        "create a tom pattern",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Tom drum pattern",
		},
		{
			name:            "cymbal_crash",
			question:        "add a crash cymbal",
			expectedDAW:     true,
			expectedDrummer: true,
			description:     "Cymbal hit",
		},
		// Non-drummer requests
		{
			name:            "chord_progression_not_drummer",
			question:        "create a chord progression",
			expectedDAW:     true,
			expectedDrummer: false,
			description:     "Should be arranger, not drummer",
		},
		{
			name:            "reverb_not_drummer",
			question:        "add reverb to track 1",
			expectedDAW:     true,
			expectedDrummer: false,
			description:     "Should be DAW only, not drummer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, needsArranger, needsDrummer := orchestrator.detectAgentsNeededKeywords(tt.question)
			detectionTime := time.Since(start)

			t.Logf("📊 Detection timing: %v", detectionTime)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Description: %s", tt.description)
			t.Logf("   Result: DAW=%v, Arranger=%v, Drummer=%v", needsDAW, needsArranger, needsDrummer)

			// Verify detection time is fast (< 1ms for keyword matching)
			assert.Less(t, detectionTime, 10*time.Millisecond,
				"Keyword detection should be very fast (<10ms)")

			// Verify expected results
			if tt.expectedDAW {
				assert.True(t, needsDAW,
					"DAW should be detected for: %q", tt.question)
			}

			if tt.expectedDrummer {
				assert.True(t, needsDrummer,
					"Drummer should be detected for: %q", tt.question)
			} else {
				assert.False(t, needsDrummer,
					"Drummer should NOT be detected for: %q", tt.question)
			}
		})
	}
}

func TestOrchestrator_Integration_DrummerWithArranger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Test that requests can trigger both drummer AND arranger
	tests := []struct {
		name        string
		question    string
		description string
	}{
		{
			name:        "drums_and_bass",
			question:    "create a drum track with a four on the floor beat and a bass track with a simple bassline",
			description: "Should trigger DAW + Drummer + Arranger",
		},
		{
			name:        "full_rhythm_section",
			question:    "create a rhythm section with drums playing a rock beat and piano playing C Am F G chords",
			description: "Full rhythm section with drums and chords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			t.Logf("Test: %s", tt.description)
			t.Logf("Question: %q", tt.question)
			t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// First check detection
			needsDAW, needsArranger, needsDrummer, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			require.NoError(t, err, "Detection should not error")
			t.Logf("🔍 Detection: DAW=%v, Arranger=%v, Drummer=%v", needsDAW, needsArranger, needsDrummer)

			// Execute
			start := time.Now()
			result, err := orchestrator.GenerateActions(ctx, tt.question, nil)
			duration := time.Since(start)

			t.Logf("⏱️  Execution time: %v", duration)

			if err != nil {
				t.Logf("❌ Error: %v", err)
				if ctx.Err() == context.DeadlineExceeded {
					t.Fatal("Test timed out")
				}
				t.Fatalf("Orchestrator error: %v", err)
			}

			require.NotNil(t, result, "Result should not be nil")
			require.NotEmpty(t, result.Actions, "Should have actions")

			// Pretty print for debugging
			actionsJSON, _ := json.MarshalIndent(result.Actions, "", "  ")
			t.Logf("📋 Actions:\n%s", string(actionsJSON))

			// Count action types
			trackCount := 0
			drumPatternCount := 0
			midiCount := 0

			for _, action := range result.Actions {
				actionType, _ := action["action"].(string)
				drumType, _ := action["type"].(string)

				if actionType == "create_track" {
					trackCount++
				}
				if drumType == "drum_pattern" || actionType == "drum_pattern" {
					drumPatternCount++
				}
				if actionType == "add_midi" {
					midiCount++
				}
			}

			t.Logf("📊 Action summary: tracks=%d, drum_patterns=%d, midi=%d",
				trackCount, drumPatternCount, midiCount)

			// Should have multiple types of actions
			assert.Greater(t, len(result.Actions), 1, "Should have multiple actions")
		})
	}
}

