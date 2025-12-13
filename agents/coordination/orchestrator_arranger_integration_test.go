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

func TestOrchestrator_Integration_ArrangerAndDAW(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		question    string
		description string
		validate    func(t *testing.T, result *OrchestratorResult)
	}{
		{
			name:        "track_with_chord_progression",
			question:    "create a new track with piano instrument and add a C Am F G chord progression",
			description: "Creates track with instrument and chord progression",
			validate: func(t *testing.T, result *OrchestratorResult) {
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				// Should have track creation action
				hasTrackCreation := false
				hasMidiAction := false
				midiNotesCount := 0

				for _, action := range result.Actions {
					actionType, ok := action["action"].(string)
					if !ok {
						continue
					}

					if actionType == "create_track" {
						hasTrackCreation = true
						// Check for instrument
						if instrument, ok := action["instrument"].(string); ok {
							assert.Contains(t, strings.ToLower(instrument), "piano",
								"Track should have piano instrument")
						}
					}

					if actionType == "add_midi" {
						hasMidiAction = true
						// Check for notes array
						if notes, ok := action["notes"].([]interface{}); ok {
							midiNotesCount = len(notes)
							assert.Greater(t, midiNotesCount, 0,
								"MIDI action should have notes")
						} else if notes, ok := action["notes"].([]map[string]any); ok {
							midiNotesCount = len(notes)
							assert.Greater(t, midiNotesCount, 0,
								"MIDI action should have notes")
						}
					}
				}

				assert.True(t, hasTrackCreation, "Should have track creation action")
				assert.True(t, hasMidiAction, "Should have MIDI action with notes")
				assert.Greater(t, midiNotesCount, 0,
					"Should have converted chord progression to MIDI notes")
			},
		},
		{
			name:        "track_with_arpeggio",
			question:    "create a new track with Serum and add an E minor arpeggio",
			description: "Creates track with instrument and arpeggio",
			validate: func(t *testing.T, result *OrchestratorResult) {
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				// Should have track creation and MIDI action with arpeggio notes
				hasTrackCreation := false
				hasMidiAction := false
				midiNotesCount := 0

				for _, action := range result.Actions {
					actionType, ok := action["action"].(string)
					if !ok {
						continue
					}

					if actionType == "create_track" {
						hasTrackCreation = true
						if instrument, ok := action["instrument"].(string); ok {
							assert.Contains(t, strings.ToLower(instrument), "serum",
								"Track should have Serum instrument")
						}
					}

					if actionType == "add_midi" {
						hasMidiAction = true
						if notes, ok := action["notes"].([]interface{}); ok {
							midiNotesCount = len(notes)
							// Arpeggio should have multiple sequential notes
							assert.GreaterOrEqual(t, midiNotesCount, 3,
								"Arpeggio should have at least 3 notes (E minor triad)")
						} else if notes, ok := action["notes"].([]map[string]any); ok {
							midiNotesCount = len(notes)
							assert.GreaterOrEqual(t, midiNotesCount, 3,
								"Arpeggio should have at least 3 notes")
						}
					}
				}

				assert.True(t, hasTrackCreation, "Should have track creation action")
				assert.True(t, hasMidiAction, "Should have MIDI action")
				assert.GreaterOrEqual(t, midiNotesCount, 3,
					"Arpeggio should have multiple sequential notes")
			},
		},
		{
			name:        "multiple_tracks_with_musical_content",
			question:    "create a piano track with a C Am F G progression and a bass track with an E minor arpeggio",
			description: "Creates multiple tracks with different musical content",
			validate: func(t *testing.T, result *OrchestratorResult) {
				require.NotNil(t, result, "Result should not be nil")
				require.NotEmpty(t, result.Actions, "Should have actions")

				// Count track creations and MIDI actions
				trackCount := 0
				midiActionCount := 0
				totalNotes := 0

				for _, action := range result.Actions {
					actionType, ok := action["action"].(string)
					if !ok {
						continue
					}

					if actionType == "create_track" {
						trackCount++
					}

					if actionType == "add_midi" {
						midiActionCount++
						if notes, ok := action["notes"].([]interface{}); ok {
							totalNotes += len(notes)
						} else if notes, ok := action["notes"].([]map[string]any); ok {
							totalNotes += len(notes)
						}
					}
				}

				assert.GreaterOrEqual(t, trackCount, 2,
					"Should create at least 2 tracks (piano and bass)")
				assert.GreaterOrEqual(t, midiActionCount, 2,
					"Should have at least 2 MIDI actions (progression and arpeggio)")
				assert.Greater(t, totalNotes, 0,
					"Should have converted musical content to MIDI notes")
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
				// Don't fail immediately - check if it's a timeout or API error
				if ctx.Err() == context.DeadlineExceeded {
					t.Fatal("Test timed out")
				}
				// For other errors, log but continue validation
				t.Logf("⚠️  Got error but continuing validation: %v", err)
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

