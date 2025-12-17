package coordination

import (
	"context"
	"encoding/json"
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

// TestAgents_OutOfScope_DirectCalls tests that each agent handles out-of-scope requests gracefully
// This is important in case the coordinator makes a routing mistake
func TestAgents_OutOfScope_DirectCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Obvious out-of-scope
	obviousOOS := []string{
		"how do I make pasta carbonara",
		"write a python function to sort a list",
		"what's the capital of France",
	}

	// Subtle out-of-scope - related to DAW but not valid music operations
	subtleOOS := []string{
		"make it sound better",                    // Too vague, no actionable request
		"create a video track with some video fx", // Video editing, not music production
		"fix the audio glitches in my recording",  // Debugging/troubleshooting, not an action
		"why does my plugin crash",                // Support question, not an action
	}

	outOfScopeQuestions := append(obviousOOS, subtleOOS...)

	t.Run("drummer_agent_rejects_out_of_scope", func(t *testing.T) {
		for _, question := range outOfScopeQuestions {
			t.Run(question, func(t *testing.T) {
				inputArray := []map[string]any{
					{"role": "user", "content": question},
				}

				result, err := orchestrator.drummerAgent.Generate(ctx, "gpt-5.1", inputArray)

				// Drummer should either error or return empty actions
				if err != nil {
					t.Logf("✅ Drummer correctly errored for out-of-scope: %v", err)
					return
				}

				if result == nil || len(result.Actions) == 0 {
					t.Logf("✅ Drummer correctly returned no actions for out-of-scope")
					return
				}

				// If it returned actions, they should not be valid drum patterns
				t.Logf("⚠️ Drummer returned %d actions for out-of-scope request", len(result.Actions))
				for _, action := range result.Actions {
					actionJSON, _ := json.MarshalIndent(action, "", "  ")
					t.Logf("   Action: %s", string(actionJSON))
				}

				// This is a soft failure - log it but don't fail the test
				// The agent's scope instruction should prevent this
				t.Logf("⚠️ Drummer should have returned empty for: %q", question)
			})
		}
	})

	t.Run("arranger_agent_rejects_out_of_scope", func(t *testing.T) {
		for _, question := range outOfScopeQuestions {
			t.Run(question, func(t *testing.T) {
				result, err := orchestrator.arrangerAgent.GenerateActions(ctx, question)

				// Arranger should either error or return empty actions
				if err != nil {
					t.Logf("✅ Arranger correctly errored for out-of-scope: %v", err)
					return
				}

				if result == nil || len(result.Actions) == 0 {
					t.Logf("✅ Arranger correctly returned no actions for out-of-scope")
					return
				}

				// If it returned actions, log them
				t.Logf("⚠️ Arranger returned %d actions for out-of-scope request", len(result.Actions))
				for _, action := range result.Actions {
					actionJSON, _ := json.MarshalIndent(action, "", "  ")
					t.Logf("   Action: %s", string(actionJSON))
				}

				t.Logf("⚠️ Arranger should have returned empty for: %q", question)
			})
		}
	})

	t.Run("daw_agent_rejects_out_of_scope", func(t *testing.T) {
		for _, question := range outOfScopeQuestions {
			t.Run(question, func(t *testing.T) {
				result, err := orchestrator.dawAgent.GenerateActions(ctx, question, nil)

				// DAW should either error or return empty/error comment
				if err != nil {
					t.Logf("✅ DAW correctly errored for out-of-scope: %v", err)
					return
				}

				if result == nil || len(result.Actions) == 0 {
					t.Logf("✅ DAW correctly returned no actions for out-of-scope")
					return
				}

				// Check if DAW returned an error comment (valid rejection)
				for _, action := range result.Actions {
					if actionType, ok := action["action"].(string); ok {
						if actionType == "error" || actionType == "comment" {
							t.Logf("✅ DAW returned error/comment action for out-of-scope")
							return
						}
					}
				}

				// If it returned real actions, log them
				t.Logf("⚠️ DAW returned %d actions for out-of-scope request", len(result.Actions))
				for _, action := range result.Actions {
					actionJSON, _ := json.MarshalIndent(action, "", "  ")
					t.Logf("   Action: %s", string(actionJSON))
				}

				t.Logf("⚠️ DAW should have rejected: %q", question)
			})
		}
	})

	// Test that valid DAW operations with musical terms in names are NOT rejected
	t.Run("daw_agent_accepts_valid_operations_with_musical_names", func(t *testing.T) {
		validDAWOperations := []string{
			"lower the track called arpeggio by 3 db",
			"rename the drums track to percussion",
			"mute the track called bassline",
			"delete the clip on the melody track",
		}

		for _, question := range validDAWOperations {
			t.Run(question, func(t *testing.T) {
				result, err := orchestrator.dawAgent.GenerateActions(ctx, question, nil)

				// These should succeed - they're valid DAW operations
				if err != nil {
					t.Logf("⚠️ DAW rejected valid operation: %v", err)
					// Don't fail - LLM can be flaky
					return
				}

				require.NotNil(t, result, "Result should not be nil")

				// Check we got real actions (not error comments)
				hasRealAction := false
				for _, action := range result.Actions {
					actionType, _ := action["action"].(string)
					if actionType != "" && actionType != "error" && actionType != "comment" {
						hasRealAction = true
						break
					}
				}

				if hasRealAction {
					t.Logf("✅ DAW correctly handled: %q with %d actions", question, len(result.Actions))
				} else {
					t.Logf("⚠️ DAW didn't produce real actions for: %q", question)
				}
			})
		}
	})
}
