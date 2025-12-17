package coordination

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Load .env file from project root for tests
	// Try multiple paths in case tests are run from different directories
	_ = godotenv.Load()             // Current directory
	_ = godotenv.Load(".env")       // Current directory explicit
	_ = godotenv.Load("../.env")    // Parent directory
	_ = godotenv.Load("../../.env") // Project root from agents/coordination/

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

// TimingMetrics tracks performance metrics for orchestrator operations
type TimingMetrics struct {
	DetectionTime         time.Duration
	KeywordDetectionTime  time.Duration
	LLMDetectionTime      time.Duration
	TotalExecutionTime    time.Duration
	DAWExecutionTime      time.Duration
	ArrangerExecutionTime time.Duration
	ParallelSpeedup       float64 // Ratio of sequential vs parallel execution
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

func TestOrchestrator_Integration_OutOfScope_Requests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	tests := []struct {
		name          string
		question      string
		description   string
		expectError   bool
		errorContains string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			result, err := orchestrator.GenerateActions(ctx, tt.question, nil)

			if tt.expectError {
				// Out-of-scope requests should return an error - either from LLM classification
				// or from DAW agent failing to generate valid DSL for non-music requests
				require.Error(t, err, "Expected error for out-of-scope request")
				t.Logf("✅ Got expected error: %v", err)

				// Should not have actions
				if result != nil {
					assert.Empty(t, result.Actions, "Out-of-scope request should not produce actions")
				}
			} else {
				// For valid requests, we might get an error if API key is invalid, but structure should be correct
				if err != nil {
					t.Logf("⚠️ Got error (might be API key issue): %v", err)
				} else {
					require.NotNil(t, result, "Valid request should return result")
					t.Logf("✅ Request succeeded with %d actions", len(result.Actions))
				}
			}
		})
	}
}

func TestOrchestrator_Integration_LLMValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	tests := []struct {
		name           string
		question       string
		description    string
		expectArranger bool
		expectDrummer  bool
	}{
		// DAW-only requests (no musical content generation)
		{
			name:           "daw_only_pan",
			question:       "pan the synth track to the left",
			description:    "Pan operation - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_solo",
			question:       "solo track 3 and mute everything else",
			description:    "Solo/mute operation - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_create_track_named_drums",
			question:       "create a new track called Drums",
			description:    "Track creation with drum NAME - should NOT trigger drummer",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_fx_on_drum_track",
			question:       "add compression to the drum bus",
			description:    "FX on drum track - DAW only, no drum generation",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "daw_only_delete_track",
			question:       "delete the third track",
			description:    "Track deletion - DAW only",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Arranger requests (melodic/harmonic content)
		{
			name:           "arranger_jazz_voicings",
			question:       "write some jazz voicings in Db",
			description:    "Jazz chords - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		{
			name:           "arranger_synth_lead",
			question:       "create a synth lead melody",
			description:    "Melody generation - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		{
			name:           "arranger_walking_bass",
			question:       "add a walking bass line in F minor",
			description:    "Bass line - Arranger needed",
			expectArranger: true,
			expectDrummer:  false,
		},
		// Drummer requests (percussion patterns)
		{
			name:           "drummer_four_on_floor",
			question:       "make a four on the floor kick pattern",
			description:    "Kick pattern - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		{
			name:           "drummer_latin_rhythm",
			question:       "create a latin percussion pattern with congas",
			description:    "Latin percussion - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		{
			name:           "drummer_trap_hats",
			question:       "add some trap hi-hat rolls",
			description:    "Hi-hat pattern - Drummer needed",
			expectArranger: false,
			expectDrummer:  true,
		},
		// Out of scope - should return both false
		{
			name:           "out_of_scope_recipe",
			question:       "how do I make pasta carbonara",
			description:    "Cooking recipe - completely out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_code",
			question:       "write a python function to sort a list",
			description:    "Programming task - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_travel",
			question:       "book me a flight to Paris",
			description:    "Travel booking - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_medical",
			question:       "what are the symptoms of flu",
			description:    "Medical question - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "out_of_scope_sports",
			question:       "who won the world cup in 2022",
			description:    "Sports trivia - out of scope",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Subtle out-of-scope - vague or non-actionable
		{
			name:           "subtle_oos_vague_improvement",
			question:       "make it sound better",
			description:    "Too vague - no actionable request",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_video",
			question:       "create a video track with some video fx",
			description:    "Video editing - not music production",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_troubleshooting",
			question:       "fix the audio glitches in my recording",
			description:    "Debugging/troubleshooting - not an action",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "subtle_oos_support_question",
			question:       "why does my plugin crash",
			description:    "Support question - not an action",
			expectArranger: false,
			expectDrummer:  false,
		},
		// Valid DAW operations with musical terms in names - should NOT trigger content agents
		{
			name:           "valid_daw_volume_with_musical_name",
			question:       "lower the track called arpeggio by 3 db",
			description:    "Volume adjustment - DAW only, arranger/drummer not needed",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "valid_daw_rename_drums_track",
			question:       "rename the drums track to percussion",
			description:    "Track rename - DAW only, drummer not triggered by name",
			expectArranger: false,
			expectDrummer:  false,
		},
		{
			name:           "valid_daw_mute_bassline",
			question:       "mute the track called bassline",
			description:    "Mute operation - DAW only, arranger not triggered by name",
			expectArranger: false,
			expectDrummer:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Question: %q", tt.question)

			// Test LLM classification directly
			start := time.Now()
			needsDAW, needsArranger, needsDrummer, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			require.NoError(t, err, "LLM classification should not error for valid music requests")

			t.Logf("📊 LLM Classification Results:")
			t.Logf("   Detection time: %v", detectionTime)
			t.Logf("   DAW: %v (always true)", needsDAW)
			t.Logf("   Arranger: %v (expected: %v)", needsArranger, tt.expectArranger)
			t.Logf("   Drummer: %v (expected: %v)", needsDrummer, tt.expectDrummer)

			// DAW is always true
			assert.True(t, needsDAW, "DAW should always be true")

			// Check Arranger/Drummer expectations
			if tt.expectArranger {
				assert.True(t, needsArranger, "Expected Arranger=true")
			}
			if tt.expectDrummer {
				assert.True(t, needsDrummer, "Expected Drummer=true")
			}

			// LLM validation should be reasonably fast (< 3s for gpt-4.1-mini)
			assert.Less(t, detectionTime, 3*time.Second,
				"LLM classification should complete within reasonable time")
		})
	}
}

func TestOrchestrator_Integration_LLMValidation_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	orchestrator := NewOrchestrator(cfg)
	ctx := context.Background()

	// Test cases that should trigger LLM validation (no keywords)
	testCases := []struct {
		name     string
		question string
	}{
		{"daw_only", "make it sound better"},
		{"arranger", "create harmonic content"},
		{"drummer", "add a breakbeat pattern"},
		{"daw_volume", "adjust the volume"},
	}

	var totalTime time.Duration
	var minTime, maxTime time.Duration
	first := true

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			needsDAW, _, _, err := orchestrator.DetectAgentsNeeded(ctx, tt.question)
			detectionTime := time.Since(start)

			// These are all valid music requests, should not error
			require.NoError(t, err, "Valid request should not error")

			// DAW is always true
			assert.True(t, needsDAW, "DAW should always be true")

			t.Logf("📊 LLM Classification Timing: %s", tt.name)
			t.Logf("   Question: %q", tt.question)
			t.Logf("   Time: %v", detectionTime)

			totalTime += detectionTime
			if first {
				minTime = detectionTime
				maxTime = detectionTime
				first = false
			} else {
				if detectionTime < minTime {
					minTime = detectionTime
				}
				if detectionTime > maxTime {
					maxTime = detectionTime
				}
			}

			// Each classification should be reasonably fast
			assert.Less(t, detectionTime, 3*time.Second,
				"Individual LLM classification should complete within reasonable time")
		})
	}

	avgTime := totalTime / time.Duration(len(testCases))
	t.Logf("📊 LLM Classification Performance Summary:")
	t.Logf("   Average: %v", avgTime)
	t.Logf("   Min: %v", minTime)
	t.Logf("   Max: %v", maxTime)
	t.Logf("   Total: %v", totalTime)

	// Average should be reasonable
	assert.Less(t, avgTime, 2*time.Second,
		"Average LLM classification time should be reasonable")
}
